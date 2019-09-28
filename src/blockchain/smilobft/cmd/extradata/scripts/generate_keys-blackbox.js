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
 * This script will generate Smilo Blackbox Keys
 *
 * eg: node generate_keys-blackbox.js 5 smilo
 *
 */

const {execSync} = require("child_process");
const fs = require('fs');

const GOPATH = process.env.GOPATH || "/opt/gocode";

const blackboxPATH = `${GOPATH}/src/Smilo-blackbox`;


if (process.argv.length < 4) {
    console.log(process.argv[1] + " <number of nodes, e.g. 7> <password, e.g. smilo>");
    process.exit(0);
}

const totalNodes = process.argv[2] * 1;
const password = process.argv[3];


const blackboxKeyNames = [];

let path = execSync("pwd").toString('ascii');
path = path.substring(0, path.length - 1);
const basePath = path + "/" + password;

const keyPath = basePath + "/keys";

for (let i = 1; i <= totalNodes; i++) {

    blackboxKeyNames.push(`${keyPath}/tm${i}`);

    if (fs.existsSync(`${blackboxKeyNames[i - 1]}.pub`)) {
        fs.unlinkSync(`${blackboxKeyNames[i - 1]}.pub`);
    }
    if (fs.existsSync(`${blackboxKeyNames[i - 1]}.key`)) {
        fs.unlinkSync(`${blackboxKeyNames[i - 1]}.key`);
    }
}


console.log("Generating blackbox keys ", `go run ${blackboxPATH}/main.go -generate-keys ${blackboxKeyNames.join()}`)
execSync(`go run ${blackboxPATH}/main.go -generate-keys ${blackboxKeyNames.join()}`, {input: "".padStart(2 * totalNodes, "\n")});
console.log(`Blackbox Keys Saved.`);
