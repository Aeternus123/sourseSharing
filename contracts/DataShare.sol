// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.19;

contract DataShare {
    struct FileInfo {
        string fileHash;
        string fileName;
        uint256 fileSize;
        address owner;
        uint256 uploadTime;
        uint256 downloadPrice;
        bool isShared;
        uint256 downloadCount;
    }
    
    struct User {
        uint256 balance;
        uint256 usedStorage;
        uint256 maxStorage;
        uint256 totalEarned;
        uint256 totalSpent;
        bool isActive;
        uint256 registerTime;
    }
    
    mapping(string => FileInfo) public files;
    mapping(address => User) public users;
    mapping(address => string[]) public userFiles;
    mapping(string => bool) public fileExists;
    
    address public admin;
    uint256 public maxUsers;
    uint256 public currentUserCount;
    bool public registrationOpen;
    
    uint256 public constant STORAGE_LIMIT = 1 * 1024 * 1024 * 1024;
    uint256 public constant UPLOAD_REWARD = 2;
    uint256 public constant DOWNLOAD_COST = 1;
    uint256 public constant FILE_DELETION_COST = 1;
    
    event FileUploaded(
        string indexed fileHash,
        string fileName,
        address indexed owner,
        uint256 fileSize,
        uint256 reward
    );
    
    event FileDownloaded(
        string indexed fileHash,
        address indexed downloader,
        address indexed owner,
        uint256 cost
    );
    
    event BalanceUpdated(
        address indexed user,
        uint256 newBalance,
        string reason
    );
    
    event UserRegistered(
        address indexed userAddress,
        address indexed registeredBy,
        uint256 timestamp
    );
    
    event RegistrationStatusChanged(
        bool registrationOpen,
        address changedBy,
        uint256 timestamp
    );
    
    event FileDeleted(
        string indexed fileHash,
        address indexed owner,
        uint256 cost
    );
    
    modifier onlyAdmin() {
        require(msg.sender == admin, "Only admin can call this function");
        _;
    }
    
    modifier onlyActiveUser() {
        // 允许管理员代表任何用户操作，或者用户自己操作
        require(msg.sender == admin || users[msg.sender].isActive, "User not active or not registered");
        _;
    }
    
    constructor(uint256 _maxUsers) {
        admin = msg.sender;
        maxUsers = _maxUsers;
        currentUserCount = 0;
        registrationOpen = false;
    }
    
    function setRegistrationStatus(bool _status) public onlyAdmin {
        registrationOpen = _status;
        emit RegistrationStatusChanged(_status, msg.sender, block.timestamp);
    }
    
    function registerUser(address _userAddress) public onlyAdmin {
        require(registrationOpen, "Registration is closed");
        require(currentUserCount < maxUsers, "Maximum user limit reached");
        require(!users[_userAddress].isActive, "User already registered");
        
        users[_userAddress] = User({
            balance: 4,
            usedStorage: 0,
            maxStorage: STORAGE_LIMIT,
            totalEarned: 0,
            totalSpent: 0,
            isActive: true,
            registerTime: block.timestamp
        });
        
        currentUserCount++;
        emit UserRegistered(_userAddress, msg.sender, block.timestamp);
    }
    
    function batchRegisterUsers(address[] memory _userAddresses) public onlyAdmin {
        require(registrationOpen, "Registration is closed");
        require(currentUserCount + _userAddresses.length <= maxUsers, "Exceeds maximum user limit");
        
        for (uint i = 0; i < _userAddresses.length; i++) {
            if (!users[_userAddresses[i]].isActive) {
                users[_userAddresses[i]] = User({
                    balance: 4,
                    usedStorage: 0,
                    maxStorage: STORAGE_LIMIT,
                    totalEarned: 0,
                    totalSpent: 0,
                    isActive: true,
                    registerTime: block.timestamp
                });
                currentUserCount++;
                emit UserRegistered(_userAddresses[i], msg.sender, block.timestamp);
            }
        }
    }
    
    function deactivateUser(address _userAddress) public onlyAdmin {
        require(users[_userAddress].isActive, "User not active");
        users[_userAddress].isActive = false;
        currentUserCount--;
    }
    
    function uploadFile(
        string memory _fileHash,
        string memory _fileName,
        uint256 _fileSize,
        bool _share
    ) public onlyActiveUser {
        // 默认使用调用者地址
        address targetUser = msg.sender;
        
        require(!fileExists[_fileHash], "File already exists");
        require(users[targetUser].usedStorage + _fileSize <= users[targetUser].maxStorage, "Storage limit exceeded");
        
        users[targetUser].usedStorage += _fileSize;
        
        files[_fileHash] = FileInfo({
            fileHash: _fileHash,
            fileName: _fileName,
            fileSize: _fileSize,
            owner: targetUser,
            uploadTime: block.timestamp,
            downloadPrice: DOWNLOAD_COST,
            isShared: _share,
            downloadCount: 0
        });
        
        fileExists[_fileHash] = true;
        userFiles[targetUser].push(_fileHash);
        
        if (_share) {
            uint256 reward = UPLOAD_REWARD + (_fileSize / (1024 * 1024));
            users[targetUser].balance += reward;
            users[targetUser].totalEarned += reward;
            
            emit FileUploaded(_fileHash, _fileName, targetUser, _fileSize, reward);
            emit BalanceUpdated(targetUser, users[targetUser].balance, "upload_reward");
        } else {
            emit FileUploaded(_fileHash, _fileName, targetUser, _fileSize, 0);
        }
    }
    
    // 管理员专用的上传文件方法，可以指定用户地址
    function adminUploadFile(
        address _userAddress,
        string memory _fileHash,
        string memory _fileName,
        uint256 _fileSize,
        bool _share
    ) public onlyAdmin {
        require(users[_userAddress].isActive, "Target user not active or not registered");
        require(!fileExists[_fileHash], "File already exists");
        require(users[_userAddress].usedStorage + _fileSize <= users[_userAddress].maxStorage, "Storage limit exceeded");
        
        users[_userAddress].usedStorage += _fileSize;
        
        files[_fileHash] = FileInfo({
            fileHash: _fileHash,
            fileName: _fileName,
            fileSize: _fileSize,
            owner: _userAddress,
            uploadTime: block.timestamp,
            downloadPrice: DOWNLOAD_COST,
            isShared: _share,
            downloadCount: 0
        });
        
        fileExists[_fileHash] = true;
        userFiles[_userAddress].push(_fileHash);
        
        if (_share) {
            uint256 reward = UPLOAD_REWARD + (_fileSize / (1024 * 1024));
            users[_userAddress].balance += reward;
            users[_userAddress].totalEarned += reward;
            
            emit FileUploaded(_fileHash, _fileName, _userAddress, _fileSize, reward);
            emit BalanceUpdated(_userAddress, users[_userAddress].balance, "upload_reward");
        } else {
            emit FileUploaded(_fileHash, _fileName, _userAddress, _fileSize, 0);
        }
    }
    
    function downloadFile(string memory _fileHash) public onlyActiveUser {
        require(fileExists[_fileHash], "File does not exist");
        
        FileInfo storage file = files[_fileHash];
        require(file.isShared, "File is not shared");
        require(users[msg.sender].balance >= file.downloadPrice, "Insufficient balance");
        
        users[msg.sender].balance -= file.downloadPrice;
        users[file.owner].balance += file.downloadPrice;
        
        users[msg.sender].totalSpent += file.downloadPrice;
        users[file.owner].totalEarned += file.downloadPrice;
        
        file.downloadCount++;
        
        emit FileDownloaded(_fileHash, msg.sender, file.owner, file.downloadPrice);
        emit BalanceUpdated(msg.sender, users[msg.sender].balance, "download_cost");
        emit BalanceUpdated(file.owner, users[file.owner].balance, "download_income");
    }
    
    function getUserInfo(address _user) public view returns (
        uint256 balance,
        uint256 usedStorage,
        uint256 maxStorage,
        uint256 totalEarned,
        uint256 totalSpent,
        bool isActive,
        uint256 registerTime
    ) {
        User storage user = users[_user];
        return (
            user.balance,
            user.usedStorage,
            user.maxStorage,
            user.totalEarned,
            user.totalSpent,
            user.isActive,
            user.registerTime
        );
    }
    
    function getFileInfo(string memory _fileHash) public view returns (
        string memory fileName,
        uint256 fileSize,
        address owner,
        uint256 uploadTime,
        uint256 downloadPrice,
        bool isShared,
        uint256 downloadCount
    ) {
        FileInfo storage file = files[_fileHash];
        return (
            file.fileName,
            file.fileSize,
            file.owner,
            file.uploadTime,
            file.downloadPrice,
            file.isShared,
            file.downloadCount
        );
    }
    
    function getUserFiles(address _user) public view returns (string[] memory) {
        return userFiles[_user];
    }
    
    function getSystemInfo() public view returns (
        uint256 _maxUsers,
        uint256 _currentUserCount,
        bool _registrationOpen,
        address _admin
    ) {
        return (
            maxUsers,
            currentUserCount,
            registrationOpen,
            admin
        );
    }
    
    function updateMaxUsers(uint256 _newMaxUsers) public onlyAdmin {
        require(_newMaxUsers >= currentUserCount, "New max users cannot be less than current user count");
        maxUsers = _newMaxUsers;
    }
    
    function deleteFile(string memory _fileHash) public onlyActiveUser {
        require(fileExists[_fileHash], "File does not exist");
        
        FileInfo storage file = files[_fileHash];
        require(file.owner == msg.sender || msg.sender == admin, "Not file owner or admin");
        require(users[msg.sender].balance >= FILE_DELETION_COST, "Insufficient balance");
        
        // 扣除删除费用
        users[msg.sender].balance -= FILE_DELETION_COST;
        users[msg.sender].totalSpent += FILE_DELETION_COST;
        
        // 更新用户存储空间
        users[file.owner].usedStorage -= file.fileSize;
        
        // 删除文件记录
        delete fileExists[_fileHash];
        delete files[_fileHash];
        
        // 从用户文件列表中删除（简化实现，不处理数组删除）
        // 在实际应用中，应该从userFiles数组中移除_fileHash
        
        emit FileDeleted(_fileHash, msg.sender, FILE_DELETION_COST);
        emit BalanceUpdated(msg.sender, users[msg.sender].balance, "file_deletion");
    }
}