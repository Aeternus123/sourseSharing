async function main() {
  console.log("开始部署合约...");
  
  const [deployer] = await ethers.getSigners();
  console.log("部署账户:", deployer.address);
  console.log("部署账户余额:", (await deployer.provider.getBalance(deployer.address)).toString());
  
  // 直接部署，跳过余额检查
  console.log("正在部署 DataShare 合约...");
  const DataShare = await ethers.getContractFactory("DataShare");
  const maxUsers = 50; // 设置更大的最大用户数
  const dataShare = await DataShare.deploy(maxUsers);
  
  // 使用新版本的等待部署方法
  await dataShare.waitForDeployment();
  
  // 使用新版本的获取地址方法
  const contractAddress = await dataShare.getAddress();
  
  console.log("✅ DataShare 合约部署成功!");
  console.log("📝 合约地址:", contractAddress);
  console.log("🔢 最大用户数:", maxUsers);
  
  // 开启注册功能
  console.log("🔄 开启注册功能...");
  await dataShare.setRegistrationStatus(true);
  console.log("✅ 注册功能已开启");
  
  // 保存合约地址
  require('fs').writeFileSync('contract-address.txt', contractAddress);
  console.log("📄 合约地址已保存到 contract-address.txt");
  
  // 显示部署信息
  console.log("\n📋 部署信息");
  console.log("----------------------------------------");
  console.log("合约地址:", contractAddress);
  console.log("管理员地址:", deployer.address);
  console.log("注册状态:", await dataShare.registrationOpen());
  console.log("----------------------------------------");
  
  console.log("\n📝 使用说明:");
  console.log("1. 请保存合约地址，用于配置服务器");
  console.log("2. 确保服务器环境变量中设置正确的合约地址");
  console.log("3. 重启服务器以使用新合约");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error("部署失败:", error);
    process.exit(1);
  });
