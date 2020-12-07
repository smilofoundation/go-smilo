echo "autonity_contract_test.go ... tests are going to be executed now"
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestCheckFeeRedirectionAndRedistribution
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestCheckBlockWithSmallFee
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestRemoveFromValidatorsList
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestAddIncorrectStakeholdersToList
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestAddStakeholderWithCorruptedEnodeToList
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestContractUpgrade_Success
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestContractUpgradeSeveralUpgrades
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestContractUpgradeSeveralUpgradesOnBusTopology
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestContractUpgradeSeveralUpgradesOnStarTopology
echo "base_test.go ... tests are going to be executed now"
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestTendermintSuccess
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestTendermintSlowConnections
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestTendermintLongRun
echo "malicious_test.go ... tests are going to be executed now"
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestTendermintOneMalicious
echo "quorum_test.go ... tests are going to be executed now"
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestTendermintNoQuorum
echo "start_stop_test.go ... tests are going to be executed now"
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestTendermintStopUpToFNodes
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestTendermintStartStopSingleNode
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestTendermintStartStopFPlusOneNodes
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestTendermintStartStopFPlusTwoNodes
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestTendermintStartStopAllNodes
echo "topology_roles_test.go ... tests are skipped"
echo "topology_test.go ... tests are going to be executed now"
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestTendermintStarSuccess
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 10m -test.run ^TestTendermintStarOverParticipantSuccess
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 20m -test.run ^TestTendermintBusSuccess
#CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 20m -test.run ^TestTendermintStartStopAllNodes
