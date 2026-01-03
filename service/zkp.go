package service

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Constants - hardcoded as per requirements
const (
	ZkpContractAddress     = "0x7587CA385f1e10c411638003dA0f1bd3C99b919e"
	MembershipQueryAddress = "0x2A152405afB201258D66919570BbD4625455a65f"
	ZkpRpcUrl              = "https://base-mainnet.infura.io/v3/64154bb5696a46eb841b6c687260559a"
	ZkpChainId             = 8453
	AllowedClubName        = "ai"
)

// ABI definitions
const zkpContractABI = `[
	{
		"inputs": [
			{"type": "uint256[2]", "name": "a"},
			{"type": "uint256[2][2]", "name": "b"},
			{"type": "uint256[2]", "name": "c"},
			{"type": "uint256[1]", "name": "input"}
		],
		"name": "verifyProof",
		"outputs": [
			{"type": "address", "name": "hashDeployer"},
			{"type": "bool", "name": "isValid"}
		],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [{"type": "bytes32", "name": "_hash"}],
		"name": "getHashStatus",
		"outputs": [
			{"type": "bool", "name": "isActive"},
			{"type": "address", "name": "deployer"},
			{"type": "bool", "name": "exists"}
		],
		"stateMutability": "view",
		"type": "function"
	}
]`

const membershipQueryABI = `[
	{
		"inputs": [
			{"type": "address", "name": "member"},
			{"type": "string", "name": "domainName"}
		],
		"name": "checkDetailedMembership",
		"outputs": [
			{"type": "bool", "name": "isPermanent"},
			{"type": "bool", "name": "isTemporary"},
			{"type": "bool", "name": "isTokenBased"},
			{"type": "bool", "name": "isCrossChain"}
		],
		"stateMutability": "view",
		"type": "function"
	}
]`

// Types
type ZkpPayload struct {
	A     [2]*big.Int
	B     [2][2]*big.Int
	C     [2]*big.Int
	Input [1]*big.Int
}

type ZkpStatus struct {
	IsActive bool
	Deployer string
	Exists   bool
}

type MembershipStatus struct {
	IsMember     bool
	IsPermanent  bool
	IsTemporary  bool
	IsTokenBased bool
	IsCrossChain bool
}

// Client singleton
var (
	ethClient     *ethclient.Client
	ethClientOnce sync.Once
	ethClientErr  error

	zkpABI        abi.ABI
	membershipABI abi.ABI
	abiOnce       sync.Once
)

func getEthClient() (*ethclient.Client, error) {
	ethClientOnce.Do(func() {
		ethClient, ethClientErr = ethclient.Dial(ZkpRpcUrl)
	})
	return ethClient, ethClientErr
}

func getABIs() (abi.ABI, abi.ABI, error) {
	var err error
	abiOnce.Do(func() {
		zkpABI, err = abi.JSON(strings.NewReader(zkpContractABI))
		if err != nil {
			return
		}
		membershipABI, err = abi.JSON(strings.NewReader(membershipQueryABI))
	})
	return zkpABI, membershipABI, err
}

// ParseZkpCode parses comma-separated zkpCode into ZkpPayload
func ParseZkpCode(code string) (*ZkpPayload, error) {
	// Remove zero-width spaces and other invisible characters
	cleanStr := strings.ReplaceAll(code, "\u200B", "")
	cleanStr = strings.ReplaceAll(cleanStr, "\u200C", "")
	cleanStr = strings.ReplaceAll(cleanStr, "\u200D", "")
	cleanStr = strings.ReplaceAll(cleanStr, "\uFEFF", "")
	cleanStr = strings.TrimSpace(cleanStr)

	parts := strings.Split(cleanStr, ",")
	if len(parts) != 9 {
		return nil, errors.New("invalid zkpCode: expected 9 comma-separated values")
	}

	// Trim each part
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	// Parse all values
	values := make([]*big.Int, 9)
	for i, part := range parts {
		// Use base 0 to auto-detect format: 0x for hex, plain numbers for decimal
		val, ok := new(big.Int).SetString(part, 0)
		if !ok {
			return nil, fmt.Errorf("invalid zkpCode: failed to parse value at index %d", i)
		}
		values[i] = val
	}

	return &ZkpPayload{
		A:     [2]*big.Int{values[0], values[1]},
		B:     [2][2]*big.Int{{values[2], values[3]}, {values[4], values[5]}},
		C:     [2]*big.Int{values[6], values[7]},
		Input: [1]*big.Int{values[8]},
	}, nil
}

// VerifyProof calls the contract to verify the proof and writes to chain
func VerifyProof(payload *ZkpPayload) (walletAddress string, txHash string, err error) {
	if common.ZkpPrivateKey == "" {
		return "", "", errors.New("ZKP_PRIVATE_KEY not configured")
	}

	client, err := getEthClient()
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to ethereum client: %w", err)
	}

	zkpABI, _, err := getABIs()
	if err != nil {
		return "", "", fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Parse private key
	privateKeyStr := common.ZkpPrivateKey
	if strings.HasPrefix(privateKeyStr, "0x") {
		privateKeyStr = privateKeyStr[2:]
	}
	privateKey, err := crypto.HexToECDSA(privateKeyStr)
	if err != nil {
		return "", "", fmt.Errorf("invalid private key: %w", err)
	}

	// Get chain ID
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	chainID := big.NewInt(ZkpChainId)

	// Create auth
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return "", "", fmt.Errorf("failed to create transactor: %w", err)
	}

	// Get nonce
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return "", "", fmt.Errorf("failed to get nonce: %w", err)
	}
	auth.Nonce = big.NewInt(int64(nonce))

	// Get gas price
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to get gas price: %w", err)
	}
	auth.GasPrice = gasPrice
	auth.GasLimit = uint64(300000) // Set a reasonable gas limit

	// Pack the call data
	contractAddr := ethcommon.HexToAddress(ZkpContractAddress)
	callData, err := zkpABI.Pack("verifyProof", payload.A, payload.B, payload.C, payload.Input)
	if err != nil {
		return "", "", fmt.Errorf("failed to pack call data: %w", err)
	}

	// First, simulate the call to get the result
	callMsg := ethereum.CallMsg{
		From: fromAddress,
		To:   &contractAddr,
		Data: callData,
	}

	result, err := client.CallContract(ctx, callMsg, nil)
	if err != nil {
		return "", "", fmt.Errorf("contract call simulation failed: %w", err)
	}

	// Unpack the result
	outputs, err := zkpABI.Unpack("verifyProof", result)
	if err != nil {
		return "", "", fmt.Errorf("failed to unpack result: %w", err)
	}

	if len(outputs) != 2 {
		return "", "", errors.New("unexpected output length from verifyProof")
	}

	hashDeployer, ok := outputs[0].(ethcommon.Address)
	if !ok {
		return "", "", errors.New("failed to parse hashDeployer from result")
	}

	isValid, ok := outputs[1].(bool)
	if !ok {
		return "", "", errors.New("failed to parse isValid from result")
	}

	if !isValid {
		return "", "", errors.New("proof verification failed")
	}

	walletAddress = hashDeployer.Hex()

	// Now send the actual transaction
	tx, err := bind.NewBoundContract(contractAddr, zkpABI, client, client, client).Transact(auth, "verifyProof", payload.A, payload.B, payload.C, payload.Input)
	if err != nil {
		return walletAddress, "", fmt.Errorf("failed to send transaction: %w", err)
	}

	txHash = tx.Hash().Hex()
	return walletAddress, txHash, nil
}

// GetHashStatus queries the zkp hash status from the contract
func GetHashStatus(zkpHash string) (*ZkpStatus, error) {
	client, err := getEthClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ethereum client: %w", err)
	}

	zkpABI, _, err := getABIs()
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Convert decimal string to bytes32 hex
	hashBigInt, ok := new(big.Int).SetString(zkpHash, 10)
	if !ok {
		return nil, errors.New("invalid zkpHash: failed to parse as decimal")
	}

	// Pad to 32 bytes
	hashBytes := make([]byte, 32)
	hashBigInt.FillBytes(hashBytes)
	var hash32 [32]byte
	copy(hash32[:], hashBytes)

	contractAddr := ethcommon.HexToAddress(ZkpContractAddress)
	callData, err := zkpABI.Pack("getHashStatus", hash32)
	if err != nil {
		return nil, fmt.Errorf("failed to pack call data: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	callMsg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: callData,
	}

	result, err := client.CallContract(ctx, callMsg, nil)
	if err != nil {
		return nil, fmt.Errorf("contract call failed: %w", err)
	}

	outputs, err := zkpABI.Unpack("getHashStatus", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %w", err)
	}

	if len(outputs) != 3 {
		return nil, errors.New("unexpected output length from getHashStatus")
	}

	isActive, _ := outputs[0].(bool)
	deployer, _ := outputs[1].(ethcommon.Address)
	exists, _ := outputs[2].(bool)

	return &ZkpStatus{
		IsActive: isActive,
		Deployer: deployer.Hex(),
		Exists:   exists,
	}, nil
}

// IsZkpValid checks if the zkp hash is still valid (not revoked)
func IsZkpValid(zkpHash string) bool {
	if zkpHash == "" {
		return true // Non-ZKP user, considered valid
	}

	status, err := GetHashStatus(zkpHash)
	if err != nil {
		common.SysLog(fmt.Sprintf("Error checking ZKP status: %v", err))
		return false // Strict mode: deny on error
	}

	return status.Exists && status.IsActive
}

// CheckClubMembership checks if the user is a member of the specified club
func CheckClubMembership(walletAddress string, clubName string) (*MembershipStatus, error) {
	client, err := getEthClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ethereum client: %w", err)
	}

	_, memberABI, err := getABIs()
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	memberAddr := ethcommon.HexToAddress(walletAddress)
	contractAddr := ethcommon.HexToAddress(MembershipQueryAddress)

	callData, err := memberABI.Pack("checkDetailedMembership", memberAddr, clubName)
	if err != nil {
		return nil, fmt.Errorf("failed to pack call data: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	callMsg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: callData,
	}

	result, err := client.CallContract(ctx, callMsg, nil)
	if err != nil {
		return nil, fmt.Errorf("contract call failed: %w", err)
	}

	outputs, err := memberABI.Unpack("checkDetailedMembership", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %w", err)
	}

	if len(outputs) != 4 {
		return nil, errors.New("unexpected output length from checkDetailedMembership")
	}

	isPermanent, _ := outputs[0].(bool)
	isTemporary, _ := outputs[1].(bool)
	isTokenBased, _ := outputs[2].(bool)
	isCrossChain, _ := outputs[3].(bool)

	return &MembershipStatus{
		IsMember:     isPermanent || isTemporary || isTokenBased || isCrossChain,
		IsPermanent:  isPermanent,
		IsTemporary:  isTemporary,
		IsTokenBased: isTokenBased,
		IsCrossChain: isCrossChain,
	}, nil
}

// IsClubMember checks if the user is a member of the allowed club
func IsClubMember(walletAddress string) bool {
	if walletAddress == "" {
		return true // Non-ZKP user, skip check
	}

	status, err := CheckClubMembership(walletAddress, AllowedClubName)
	if err != nil {
		common.SysLog(fmt.Sprintf("Error checking club membership: %v", err))
		return false // Strict mode: deny on error
	}

	return status.IsMember
}
