package core

import (
    "math/big"
    "time"
    
    "github.com/ethereum/go-ethereum/common"
)

type UserInfo struct {
    Address      string   `json:"address"`
    Balance      *big.Int `json:"balance"`
    UsedStorage  uint64   `json:"used_storage"`
    MaxStorage   uint64   `json:"max_storage"`
    TotalEarned  *big.Int `json:"total_earned"`
    TotalSpent   *big.Int `json:"total_spent"`
    IsActive     bool     `json:"is_active"`
    RegisterTime uint64   `json:"register_time"`
}

type BlockchainFileInfo struct {
    FileHash      string   `json:"file_hash"`
    FileName      string   `json:"file_name"`
    FileSize      uint64   `json:"file_size"`
    Owner         string   `json:"owner"`
    UploadTime    uint64   `json:"upload_time"`
    DownloadPrice *big.Int `json:"download_price"`
    IsShared      bool     `json:"is_shared"`
    DownloadCount uint64   `json:"download_count"`
}

type LocalFileInfo struct {
    FileHash   string    `json:"file_hash"`
    FileName   string    `json:"file_name"`
    FileSize   int64     `json:"file_size"`
    FilePath   string    `json:"file_path"`
    Owner      string    `json:"owner"`
    UploadTime time.Time `json:"upload_time"`
    IsShared   bool      `json:"is_shared"`
}

type UploadRequest struct {
    UserID   string `json:"user_id" binding:"required"`
    FileName string `json:"file_name" binding:"required"`
    FileSize int64  `json:"file_size" binding:"required"`
    Share    bool   `json:"share"`
}

type DownloadRequest struct {
    Downloader string `json:"downloader" binding:"required"`
    Owner      string `json:"owner" binding:"required"`
    FileHash   string `json:"file_hash" binding:"required"`
}

type TransactionResponse struct {
    TxHash    string `json:"tx_hash"`
    Status    string `json:"status"`
    Message   string `json:"message"`
    Timestamp int64  `json:"timestamp"`
}

type SystemInfo struct {
    MaxUsers         uint64 `json:"max_users"`
    CurrentUserCount uint64 `json:"current_user_count"`
    RegistrationOpen bool   `json:"registration_open"`
    AdminAddress     string `json:"admin_address"`
    TotalStorageUsed uint64 `json:"total_storage_used"`
    TotalStorageLimit uint64 `json:"total_storage_limit"`
}

type RegisterUserRequest struct {
    UserAddress string `json:"user_address" binding:"required"`
}

type BatchRegisterRequest struct {
    UserAddresses []string `json:"user_addresses" binding:"required"`
}

type RegistrationStatusRequest struct {
    Open bool `json:"open" binding:"required"`
}

type AdminAuth struct {
    AdminAddress string `json:"admin_address" binding:"required"`
    Signature    string `json:"signature" binding:"required"`
}

func GetCurrentTimestamp() int64 {
    return time.Now().Unix()
}

func IsValidAddress(address string) bool {
    return common.IsHexAddress(address)
}