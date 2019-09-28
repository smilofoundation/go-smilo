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
 *
 * eg: node generate-nodesList.js 5 smilo
 *
 */

const {execSync} = require("child_process");
const fs = require('fs');

const GOPATH = process.env.GOPATH || "/opt/gocode";

const smiloPATH = `${GOPATH}/src/go-smilo/src/blockchain/smilobft`;


if (process.argv.length < 4) {
    console.log(process.argv[1] + " <number of nodes, e.g. 7> <password, e.g. smilo>");
    process.exit(1);
}


const password = process.argv[3];


const totalNodes = process.argv[2] * 1;
let path = execSync("pwd").toString('ascii');
path = path.substring(0, path.length - 1);

const basePath = path + "/" + password;

const nodePath = basePath + "/nodekeys";

const keyPath = basePath + "/keys";

if (!fs.existsSync(keyPath)) {
    console.log(`Could not find the proper keyPath, verify that ${keyPath} exists`);
    process.exit(1);
}

if (!fs.existsSync(nodePath)) {
    console.log(`Could not find the proper nodePath, verify that ${nodePath} exists`);
    process.exit(1);
}

const enodeURLs = [];

for (let i = 1; i <= totalNodes; i++) {

    const keyFileName = `${nodePath}/nodekey${i}`;

    const privateKey = fs.readFileSync(keyFileName, "utf8");

    const enode = execSync(`go run ${smiloPATH}/cmd/bootnode/main.go -v5 -nodekeyhex ${privateKey} -writeaddress`).toString('utf8').replace('\n', '');

    enodeURLs.push(`enode://${enode}@127.0.0.1:${20999 + i}?discport=0`);

    console.log(`Key ${i} readed.`);

}


fs.writeFileSync(basePath + "/permissioned-nodes.json", JSON.stringify(enodeURLs, null, 2));
fs.writeFileSync(basePath + "/static-nodes.json", JSON.stringify(enodeURLs, null, 2));

