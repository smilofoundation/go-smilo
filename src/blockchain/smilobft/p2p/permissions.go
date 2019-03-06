package p2p

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/log"

	"go-smilo/src/blockchain/smilobft/p2p/enode"
)

const (
	NODE_NAME_LENGTH    = 32
	PERMISSIONED_CONFIG = "permissioned-nodes.json"
)

// check if a given node is permissioned to connect to the change
func IsNodePermissioned(nodeID string, currentNode string, datadir string, direction string) bool {

	var permissionedList []string
	nodes := ParsePermissionedNodes(datadir)
	for _, v := range nodes {
		permissionedList = append(permissionedList, v.ID().String())
	}

	log.Trace("isNodePermissioned", "permissionedList", permissionedList)
	for _, v := range permissionedList {
		if v == nodeID {
			log.Trace("isNodePermissioned", "connection", direction, "nodename", nodeID[:NODE_NAME_LENGTH], "ALLOWED-BY", currentNode[:NODE_NAME_LENGTH])
			return true
		}
	}
	log.Trace("isNodePermissioned", "connection", direction, "nodename", nodeID[:NODE_NAME_LENGTH], "DENIED-BY", currentNode[:NODE_NAME_LENGTH])
	return false
}

func ParsePermissionedNodes(DataDir string) []*enode.Node {

	log.Trace("parsePermissionedNodes", "DataDir", DataDir, "file", PERMISSIONED_CONFIG)

	path := filepath.Join(DataDir, PERMISSIONED_CONFIG)
	if _, err := os.Stat(path); err != nil {
		log.Error("Read Error for permissioned-nodes.json file. This is because 'permissioned' flag is specified but no permissioned-nodes.json file is present.", "err", err)
		return nil
	}
	// Load the nodes from the config file
	blob, err := ioutil.ReadFile(path)
	if err != nil {
		log.Error("parsePermissionedNodes: Failed to access nodes", "err", err)
		return nil
	}

	nodelist := []string{}
	if err := json.Unmarshal(blob, &nodelist); err != nil {
		log.Error("parsePermissionedNodes: Failed to load nodes", "err", err)
		return nil
	}
	// Interpret the list as a discovery node array
	var nodes []*enode.Node
	for _, url := range nodelist {
		if url == "" {
			log.Error("parsePermissionedNodes: Node URL blank")
			continue
		}
		node, err := enode.ParseV4(url)
		if err != nil {
			log.Error("parsePermissionedNodes: Node URL", "url", url, "err", err)
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes
}
