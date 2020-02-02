package core

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/log"
)

const (
	BLACKLIST_CONFIG = "blacklisted-addresses.json"
)

// check if a given address is blacklisted
func IsAddressBlacklisted(address string, blacklistFile string) bool {
	if blacklistFile == "" {
		return false
	}
	addresses := ParseBlacklistedAddresses(blacklistFile)
	if addresses != nil {
		log.Debug("IsAddressBlacklisted", "address", address, "addresses", addresses)
		for _, v := range addresses {
			if strings.ToLower(v) == strings.ToLower(address) {
				log.Debug("IsAddressBlacklisted true", "address", address)
				return true
			}
		}
	}
	log.Debug("IsAddressBlacklisted false", "address", address)
	return false
}

func ParseBlacklistedAddresses(blacklistFile string) (nodelist []string) {
	log.Trace("ParseBlacklistedAddresses", "blacklistFile", blacklistFile, "file", BLACKLIST_CONFIG)
	if _, err := os.Stat(blacklistFile); err != nil {
		log.Error("Read Error for blacklisted-addresses.json file. This is because 'blacklist' flag is specified but no blacklisted-addresses.json file is present.", "err", err)
		return nil
	}
	// Load the addresses from the config file
	blob, err := ioutil.ReadFile(blacklistFile)
	if err != nil {
		log.Error("ParseBlacklistedAddresses: Failed to access list of blacklisted addresses", "err", err)
		return nil
	}
	if err := json.Unmarshal(blob, &nodelist); err != nil {
		log.Error("ParseBlacklistedAddresses: Failed to load addresses", "err", err)
		return nil
	}
	return nodelist
}
