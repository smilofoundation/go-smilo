CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 20m -test.run ^TestTendermintStartStopFPlusOneNodes
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 20m -test.run ^TestTendermintStartStopFPlusTwoNodes
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 20m -test.run ^TestTendermintStartStopAllNodes
echo "topology_roles_test.go ... tests are skipped"
echo "topology_test.go ... tests are going to be executed now"
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 20m -test.run ^TestTendermintStarSuccess
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 20m -test.run ^TestTendermintStarOverParticipantSuccess
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 20m -test.run ^TestTendermintBusSuccess
CONSENSUS_TEST_MODE=tendermint go test ./src/blockchain/smilobft/consensus/test/... --count=1 -timeout 20m -test.run ^TestTendermintStartStopAllNodes
