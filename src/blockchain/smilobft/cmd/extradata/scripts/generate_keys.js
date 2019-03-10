// Copyright 2019 The go-smilo Authors
// This file is part of the go-smilo library.
//
// The go-smilo library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-smilo library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-smilo library. If not, see <http://www.gnu.org/licenses/>.

/**
 *
 * This script will generate ETH keys + Smilo Blackbox Keys and glue all together on the Smilo Genesis json
 *
 * eg: rm -rf smilo && node generate_keys.js 5 smilo
 *
 */

const {execSync} = require("child_process");
const fs = require('fs');
const Wallet = require('ethereumjs-wallet');
const EthUtil = require('ethereumjs-util');


// const smiloPATH = "/opt/go-code/src/go-smilo/src/blockchain/smilobft";
// const blackboxPATH = "/opt/go-code/src/Smilo-blackbox";

const smiloPATH = "../../../";
const blackboxPATH = "../../../../../../../Smilo-blackbox";


if (process.argv.length < 4) {
    console.log(process.argv[1] + " <number of nodes, e.g. 7> <password, e.g. smilo>");
    process.exit(0);
}


const password = process.argv[3];
const totalNodes = process.argv[2] * 1;
let path = execSync("pwd").toString('ascii');
path = path.substring(0, path.length - 1);

const privateKeyPrefix = password.padStart(32, "\0");
const basePath = path + "/" + password;

if (!fs.existsSync(basePath)) {
    fs.mkdirSync(basePath);
}

fs.writeFileSync(basePath + "/passwords.txt", password);

const keyPath = basePath + "/keys";

if (!fs.existsSync(keyPath)) {
    fs.mkdirSync(keyPath);
}

const nodePath = basePath + "/nodekeys";

if (!fs.existsSync(nodePath)) {
    fs.mkdirSync(nodePath);
}

const blackboxKeyNames = [];
const enodeURLs = [];
const addressList = [];
const addressListStriped = [];

for (let i = 1; i <= totalNodes; i++) {

    // const privateKey = (privateKeyPrefix + i).slice(-32);
    const wallet = Wallet.generate();
    const publicKey = wallet.getPublicKeyString();
    const privateKey = wallet.getPrivateKeyString().slice(-64);

    const address = wallet.getAddressString();
    addressList.push(address);
    addressListStriped.push(address.substring(2));

    const keystore = wallet.toV3(password);

    const pk_info = `${keyPath}/key${i}.info`;
    const key1 = `${keyPath}/key${i}`;
    const nodekey1 = `${nodePath}/nodekey${i}`;

    fs.writeFileSync(pk_info, `Private Key: ${wallet.getPrivateKeyString()}\nPublic Key: ${publicKey}\nAddress: ${address}\n`);

    fs.writeFileSync(key1, JSON.stringify(keystore, null, 2));

    fs.writeFileSync(nodekey1, privateKey);

    const enode = execSync(`go run ${smiloPATH}/cmd/bootnode/main.go -v5 -nodekeyhex ${privateKey} -writeaddress`).toString('utf8').replace('\n', '');

    enodeURLs.push(`enode://${enode}@127.0.0.1:${20999 + i}?discport=0`);

    console.log(`Key ${i} Saved.`);

    blackboxKeyNames.push(`${keyPath}/tm${i}`);

    if (fs.existsSync(`${blackboxKeyNames[i - 1]}.pub`)) {
        fs.unlinkSync(`${blackboxKeyNames[i - 1]}.pub`);
    }
    if (fs.existsSync(`${blackboxKeyNames[i - 1]}.key`)) {
        fs.unlinkSync(`${blackboxKeyNames[i - 1]}.key`);
    }
}


// console.log("Generating blackbox keys ", `go run ${blackboxPATH}/main.go -generate-keys -filename ${blackboxKeyNames.join()}`)
execSync(`go run ${blackboxPATH}/main.go -generate-keys -filename ${blackboxKeyNames.join()}`, {input: "".padStart(2 * totalNodes, "\n")});
console.log(`Blackbox Keys Saved.`);

fs.writeFileSync(basePath + "/permissioned-nodes.json", JSON.stringify(enodeURLs, null, 2));

// console.log("Running extra data encode for fullnodes ", `go run ${smiloPATH}/cmd/extradata/main.go`)
const extraDataCmdResult = execSync(`go run ${smiloPATH}/cmd/extradata/main.go extra encode -fullnodes ${addressList.join()}`);
let extraData = "";

if (extraDataCmdResult) {
    const extraDataUTF = extraDataCmdResult.toString('utf8');
    const extraDataSubstring = extraDataUTF.substring("Encoded Sport extra-data: ".length);
    extraData = extraDataSubstring.slice(0, -1);
    // console.log("*"+extraData+"*");
} else {
    throw new Error("Could not generate extraData for the encoded list of fullnodes")
}

const genesisString = "{  \"alloc\": { \"" + addressListStriped.join("\": {      \"balance\": \"0x446c3b15f9926687d2c40534fdb564000000000000\"    },    \"") +
    "\": {      \"balance\": \"0x446c3b15f9926687d2c40534fdb564000000000000\"    }" +
    "  },  \"coinbase\": \"0x0000000000000000000000000000000000000000\",  \"config\": {    \"byzantiumBlock\": 1,    \"eip150Block\": 2,    \"eip150Hash\": \"0x0000000000000000000000000000000000000000000000000000000000000000\",    \"eip155Block\": 0,    \"eip158Block\": 3,    \"petersburgBlock\": 4,    \"constantinopleBlock\": 5,    \"sport\": {      \"epoch\": 30000,      \"policy\": 0    },   " +
    " \"isSmilo\": true,  \"isGas\": true,  \"isGasRefunded\": true,  \"chainId\": 10" +
    "  },  \"extraData\": \"" + extraData + "\",  \"gasLimit\": \"0x2518C7E00\",  \"difficulty\": \"0x1\",  \"mixHash\": \"0x636861696e20706c6174666f726d2077697468206120636f6e736369656e6365\",  \"nonce\": \"0x0\",  \"parentHash\": \"0x0000000000000000000000000000000000000000000000000000000000000000\",  \"timestamp\": \"0x00\"}";


const genesis = JSON.parse(genesisString);

fs.writeFileSync(basePath + "/smilo-genesis.json", JSON.stringify(genesis, null, 2));