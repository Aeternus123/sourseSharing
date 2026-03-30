package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
)

// TransactionRecord 交易记录结构
type TransactionRecord struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"` // "upload", "download", "balance_update", "user_register"
	TxHash        string    `json:"tx_hash"`
	BlockNumber   uint64    `json:"block_number"`
	Timestamp     uint64    `json:"timestamp"`
	From          string    `json:"from"`
	To            string    `json:"to,omitempty"`
	Amount        string    `json:"amount,omitempty"`
	FileHash      string    `json:"file_hash,omitempty"`
	FileName      string    `json:"file_name,omitempty"`
	FileSize      uint64    `json:"file_size,omitempty"`
	UserAddress   string    `json:"user_address,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// Ledger 账本管理器
type Ledger struct {
	ethClient      *ethclient.Client
	contract       *DataShare
	dbPath         string
	records        []*TransactionRecord
	recordsMutex   sync.RWMutex
	eventChan      chan *TransactionRecord
	stopChan       chan struct{}
	isRunning      bool
	startBlock     uint64
}

// NewLedger 创建新的账本管理器
func NewLedger(client *ethclient.Client, contract *DataShare, dbPath string) *Ledger {
	return &Ledger{
		ethClient:    client,
		contract:     contract,
		dbPath:       dbPath,
		records:      make([]*TransactionRecord, 0),
		eventChan:    make(chan *TransactionRecord, 100),
		stopChan:     make(chan struct{}),
		isRunning:    false,
		startBlock:   0,
	}
}

// Init 初始化账本
func (l *Ledger) Init() error {
	// 创建数据目录
	if err := os.MkdirAll(l.dbPath, 0755); err != nil {
		return fmt.Errorf("创建账本目录失败: %v", err)
	}

	// 加载已有记录
	if err := l.loadRecords(); err != nil {
		log.Printf("警告: 加载账本记录失败: %v，将从新区块开始同步", err)
	}

	return nil
}

// Start 启动事件监听
func (l *Ledger) Start() error {
	if l.isRunning {
		return nil
	}

	l.isRunning = true
	go l.startEventListeners()
	go l.processEvents()

	log.Printf("✅ 账本事件监听已启动，从区块 %d 开始同步", l.startBlock)
	return nil
}

// Stop 停止事件监听
func (l *Ledger) Stop() {
	if !l.isRunning {
		return
	}

	close(l.stopChan)
	l.isRunning = false
	log.Println("✅ 账本事件监听已停止")
}

// GetAllRecords 获取所有交易记录
func (l *Ledger) GetAllRecords() []*TransactionRecord {
	l.recordsMutex.RLock()
	defer l.recordsMutex.RUnlock()

	// 返回副本
	recordsCopy := make([]*TransactionRecord, len(l.records))
	copy(recordsCopy, l.records)
	return recordsCopy
}

// GetRecordsByUser 获取特定用户的交易记录
func (l *Ledger) GetRecordsByUser(userAddress string) []*TransactionRecord {
	l.recordsMutex.RLock()
	defer l.recordsMutex.RUnlock()

	var userRecords []*TransactionRecord
	for _, record := range l.records {
		if record.From == userAddress || record.To == userAddress || record.UserAddress == userAddress {
			userRecords = append(userRecords, record)
		}
	}
	return userRecords
}

// GetRecordsByType 获取特定类型的交易记录
func (l *Ledger) GetRecordsByType(recordType string) []*TransactionRecord {
	l.recordsMutex.RLock()
	defer l.recordsMutex.RUnlock()

	var typeRecords []*TransactionRecord
	for _, record := range l.records {
		if record.Type == recordType {
			typeRecords = append(typeRecords, record)
		}
	}
	return typeRecords
}

// startEventListeners 启动所有事件监听器
func (l *Ledger) startEventListeners() {
	// 监听文件上传事件
	go l.listenFileUploadedEvents()

	// 监听文件下载事件
	go l.listenFileDownloadedEvents()

	// 监听余额更新事件
	go l.listenBalanceUpdatedEvents()

	// 监听用户注册事件
	go l.listenUserRegisteredEvents()
}

// listenFileUploadedEvents 监听FileUploaded事件
func (l *Ledger) listenFileUploadedEvents() {
	// 获取最新区块号
	latestBlock, err := l.ethClient.BlockNumber(context.Background())
	if err != nil {
		log.Printf("获取最新区块失败: %v，将使用默认区块 %d", err, l.startBlock)
		latestBlock = l.startBlock
	}

	// 创建事件过滤器
	filterOpts := &bind.FilterOpts{
		Start:   l.startBlock,
		End:     &latestBlock,
		Context: context.Background(),
	}

	// 持续监听
	for {
		select {
		case <-l.stopChan:
			return
		default:
			iter, err := l.contract.FilterFileUploaded(filterOpts, nil, nil)
			if err != nil {
				log.Printf("创建FileUploaded事件过滤器失败: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			// 处理现有事件
			for iter.Next() {
				event := iter.Event
				record := &TransactionRecord{
					ID:          fmt.Sprintf("upload_%s", event.FileHash),
					Type:        "upload",
					TxHash:      iter.Event.Raw.TxHash.Hex(),
					BlockNumber: iter.Event.Raw.BlockNumber,
					Timestamp:   uint64(time.Now().Unix()),
					From:        event.Owner.Hex(),
					FileHash:    event.FileHash.Hex(),
					FileName:    event.FileName,
					FileSize:    event.FileSize.Uint64(),
					Amount:      event.Reward.String(),
					CreatedAt:   time.Now(),
				}

				l.eventChan <- record
			}

			iter.Close()

			// 更新起始区块，只监听新区块
			l.startBlock = latestBlock + 1
			latestBlock, err = l.ethClient.BlockNumber(context.Background())
			if err != nil {
				log.Printf("更新区块号失败: %v", err)
			}

			time.Sleep(3 * time.Second)
		}
	}
}

// listenFileDownloadedEvents 监听FileDownloaded事件
func (l *Ledger) listenFileDownloadedEvents() {
	// 获取最新区块号
	latestBlock, err := l.ethClient.BlockNumber(context.Background())
	if err != nil {
		log.Printf("获取最新区块失败: %v，将使用默认区块 %d", err, l.startBlock)
		latestBlock = l.startBlock
	}

	// 创建事件过滤器
	filterOpts := &bind.FilterOpts{
		Start:   l.startBlock,
		End:     &latestBlock,
		Context: context.Background(),
	}

	// 持续监听
	for {
		select {
		case <-l.stopChan:
			return
		default:
			iter, err := l.contract.FilterFileDownloaded(filterOpts, []string{}, nil, nil)
			if err != nil {
				log.Printf("创建FileDownloaded事件过滤器失败: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			// 处理现有事件
			for iter.Next() {
				event := iter.Event
				record := &TransactionRecord{
					ID:          fmt.Sprintf("download_%s_%d", event.FileHash, time.Now().Unix()),
					Type:        "download",
					TxHash:      iter.Event.Raw.TxHash.Hex(),
					BlockNumber: iter.Event.Raw.BlockNumber,
					Timestamp:   uint64(time.Now().Unix()),
					From:        event.Downloader.Hex(),
					To:          event.Owner.Hex(),
					FileHash:    event.FileHash.Hex(),
					Amount:      event.Cost.String(),
					CreatedAt:   time.Now(),
				}

				l.eventChan <- record
			}

			iter.Close()

			time.Sleep(3 * time.Second)
		}
	}
}

// listenBalanceUpdatedEvents 监听BalanceUpdated事件
func (l *Ledger) listenBalanceUpdatedEvents() {
	// 获取最新区块号
	latestBlock, err := l.ethClient.BlockNumber(context.Background())
	if err != nil {
		log.Printf("获取最新区块失败: %v，将使用默认区块 %d", err, l.startBlock)
		latestBlock = l.startBlock
	}

	// 创建事件过滤器
	filterOpts := &bind.FilterOpts{
		Start:   l.startBlock,
		End:     &latestBlock,
		Context: context.Background(),
	}

	// 持续监听
	for {
		select {
		case <-l.stopChan:
			return
		default:
			iter, err := l.contract.FilterBalanceUpdated(filterOpts, nil)
			if err != nil {
				log.Printf("创建BalanceUpdated事件过滤器失败: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			// 处理现有事件
			for iter.Next() {
				event := iter.Event
				record := &TransactionRecord{
					ID:          fmt.Sprintf("balance_%s_%d", event.User.Hex(), time.Now().Unix()),
					Type:        "balance_update",
					TxHash:      iter.Event.Raw.TxHash.Hex(),
					BlockNumber: iter.Event.Raw.BlockNumber,
					Timestamp:   uint64(time.Now().Unix()),
					UserAddress: event.User.Hex(),
					Amount:      event.NewBalance.String(),
					CreatedAt:   time.Now(),
				}

				l.eventChan <- record
			}

			iter.Close()

			time.Sleep(3 * time.Second)
		}
	}
}

// listenUserRegisteredEvents 监听UserRegistered事件
func (l *Ledger) listenUserRegisteredEvents() {
	// 获取最新区块号
	latestBlock, err := l.ethClient.BlockNumber(context.Background())
	if err != nil {
		log.Printf("获取最新区块失败: %v，将使用默认区块 %d", err, l.startBlock)
		latestBlock = l.startBlock
	}

	// 创建事件过滤器
	filterOpts := &bind.FilterOpts{
		Start:   l.startBlock,
		End:     &latestBlock,
		Context: context.Background(),
	}

	// 持续监听
	for {
		select {
		case <-l.stopChan:
			return
		default:
			iter, err := l.contract.FilterUserRegistered(filterOpts, nil, nil)
			if err != nil {
				log.Printf("创建UserRegistered事件过滤器失败: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			// 处理现有事件
			for iter.Next() {
				event := iter.Event
				record := &TransactionRecord{
					ID:          fmt.Sprintf("register_%s", event.Raw.TxHash.Hex()),
					Type:        "user_register",
					TxHash:      iter.Event.Raw.TxHash.Hex(),
					BlockNumber: iter.Event.Raw.BlockNumber,
					Timestamp:   uint64(time.Now().Unix()),
					UserAddress: event.UserAddress.Hex(),
					CreatedAt:   time.Now(),
				}

				l.eventChan <- record
			}

			iter.Close()

			time.Sleep(3 * time.Second)
		}
	}
}

// processEvents 处理事件通道中的事件
func (l *Ledger) processEvents() {
	for {
		select {
		case <-l.stopChan:
			return
		case record := <-l.eventChan:
			l.addRecord(record)
			log.Printf("✅ 记录新交易: %s - 类型: %s, 用户: %s", record.ID, record.Type, record.From)
		}
	}
}

// addRecord 添加交易记录
func (l *Ledger) addRecord(record *TransactionRecord) {
	l.recordsMutex.Lock()
	defer l.recordsMutex.Unlock()

	// 检查是否已存在相同ID的记录
	for _, r := range l.records {
		if r.ID == record.ID && r.Type != "download" { // 下载记录允许重复
			return
		}
	}

	l.records = append(l.records, record)
	
	// 异步保存到文件
	go l.saveRecords()
}

// loadRecords 从文件加载交易记录
func (l *Ledger) loadRecords() error {
	filePath := filepath.Join(l.dbPath, "ledger.json")
	file, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var records []*TransactionRecord
	if err := json.Unmarshal(file, &records); err != nil {
		return err
	}

	l.recordsMutex.Lock()
	defer l.recordsMutex.Unlock()
	l.records = records

	// 找到最大的区块号，用于后续同步
	maxBlock := uint64(0)
	for _, record := range records {
		if record.BlockNumber > maxBlock {
			maxBlock = record.BlockNumber
		}
	}
	l.startBlock = maxBlock

	log.Printf("✅ 已加载 %d 条账本记录，最大区块号: %d", len(records), maxBlock)
	return nil
}

// saveRecords 保存交易记录到文件
func (l *Ledger) saveRecords() {
	l.recordsMutex.RLock()
	recordsCopy := make([]*TransactionRecord, len(l.records))
	copy(recordsCopy, l.records)
	l.recordsMutex.RUnlock()

	filePath := filepath.Join(l.dbPath, "ledger.json")
	data, err := json.MarshalIndent(recordsCopy, "", "  ")
	if err != nil {
		log.Printf("保存账本记录失败: %v", err)
		return
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		log.Printf("写入账本文件失败: %v", err)
		return
	}

	log.Printf("✅ 账本已保存，共 %d 条记录", len(recordsCopy))
}

// GetSyncInfo 获取同步状态信息
func (l *Ledger) GetSyncInfo() map[string]interface{} {
	latestBlock, _ := l.ethClient.BlockNumber(context.Background())
	
	return map[string]interface{}{
		"is_syncing":      l.isRunning,
		"current_block":   l.startBlock,
		"latest_block":    latestBlock,
		"records_count":   len(l.records),
		"sync_progress":   fmt.Sprintf("%.2f%%", float64(l.startBlock)/float64(latestBlock+1)*100),
	}
}