package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"
)

type FileStorage struct {
	BasePath string
}

func NewFileStorage(basePath string) *FileStorage {
	if basePath == "" {
		basePath = "./storage"
	}
	os.MkdirAll(basePath, 0755)
	return &FileStorage{BasePath: basePath}
}

func (fs *FileStorage) SaveFile(userID string, fileHeader *multipart.FileHeader) (*LocalFileInfo, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, fmt.Errorf("计算文件哈希失败: %v", err)
	}
	fileHash := hex.EncodeToString(hasher.Sum(nil))

	file.Seek(0, 0)

	userDir := filepath.Join(fs.BasePath, userID)
	os.MkdirAll(userDir, 0755)

	filePath := filepath.Join(userDir, fileHash)
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return nil, fmt.Errorf("保存文件失败: %v", err)
	}

	return &LocalFileInfo{
		FileHash:   fileHash,
		FileName:   fileHeader.Filename,
		FileSize:   fileHeader.Size,
		FilePath:   filePath,
		Owner:      userID,
		UploadTime: time.Now(),
		IsShared:   false,
	}, nil
}

func (fs *FileStorage) SaveFileBytes(userID string, fileName string, fileData []byte) (*LocalFileInfo, error) {
	hasher := sha256.New()
	hasher.Write(fileData)
	fileHash := hex.EncodeToString(hasher.Sum(nil))

	userDir := filepath.Join(fs.BasePath, userID)
	os.MkdirAll(userDir, 0755)

	filePath := filepath.Join(userDir, fileHash)
	err := os.WriteFile(filePath, fileData, 0644)
	if err != nil {
		return nil, fmt.Errorf("保存文件失败: %v", err)
	}

	return &LocalFileInfo{
		FileHash:   fileHash,
		FileName:   fileName,
		FileSize:   int64(len(fileData)),
		FilePath:   filePath,
		Owner:      userID,
		UploadTime: time.Now(),
		IsShared:   false,
	}, nil
}

func (fs *FileStorage) GetFile(userID, fileHash string) (string, error) {
	filePath := filepath.Join(fs.BasePath, userID, fileHash)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("文件不存在")
	}

	return filePath, nil
}

func (fs *FileStorage) GetFileInfo(userID, fileHash string) (*LocalFileInfo, error) {
	filePath := filepath.Join(fs.BasePath, userID, fileHash)

	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("文件不存在")
	}
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %v", err)
	}

	return &LocalFileInfo{
		FileHash:   fileHash,
		FileName:   filepath.Base(filePath),
		FileSize:   fileInfo.Size(),
		FilePath:   filePath,
		Owner:      userID,
		UploadTime: fileInfo.ModTime(),
		IsShared:   false,
	}, nil
}

func (fs *FileStorage) DeleteFile(userID, fileHash string) error {
	filePath := filepath.Join(fs.BasePath, userID, fileHash)
	return os.Remove(filePath)
}

func (fs *FileStorage) CalculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (fs *FileStorage) GetUserFiles(userID string) ([]*LocalFileInfo, error) {
	userDir := filepath.Join(fs.BasePath, userID)

	if _, err := os.Stat(userDir); os.IsNotExist(err) {
		return []*LocalFileInfo{}, nil
	}

	files, err := os.ReadDir(userDir)
	if err != nil {
		return nil, fmt.Errorf("读取用户目录失败: %v", err)
	}

	var fileInfos []*LocalFileInfo
	for _, file := range files {
		if !file.IsDir() {
			filePath := filepath.Join(userDir, file.Name())
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				continue
			}

			fileInfos = append(fileInfos, &LocalFileInfo{
				FileHash:   file.Name(),
				FileName:   file.Name(),
				FileSize:   fileInfo.Size(),
				FilePath:   filePath,
				Owner:      userID,
				UploadTime: fileInfo.ModTime(),
				IsShared:   false,
			})
		}
	}

	return fileInfos, nil
}
