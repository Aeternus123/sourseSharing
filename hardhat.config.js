require("@nomicfoundation/hardhat-toolbox");

/** @type import('hardhat/config').HardhatConfig */
module.exports = {
  solidity: "0.8.19",
  networks: {
    hardhat: {
      chainId: 31337
    },
    localhost: {
      url: "http://0.0.0.0:8545",
      chainId: 31337
    }
  }
};