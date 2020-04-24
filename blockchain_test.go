package bc

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func InitForTest(numOfPeers, numOfValidators int) ([]*Node, error) {

	genesis := Genesis{
		make(map[string]uint64),
		make([]crypto.PublicKey, 0, numOfPeers),
	}

	peers := make([]*Node, numOfPeers)
	keys := make([]ed25519.PrivateKey, numOfPeers)
	for i := range keys {
		_, key, err := ed25519.GenerateKey(nil)
		if err != nil {
			return nil, err
		}
		keys[i] = key
		if numOfValidators > 0 {
			genesis.Validators = append(genesis.Validators, key.Public())
			numOfValidators--
		}

		address, err := PubKeyToAddress(key.Public())
		if err != nil {
			return nil, err
		}
		genesis.Alloc[address] = 1000
	}

	var err error
	for i := 0; i < 3; i++  {
		peers[i], err = NewNode(keys[i], genesis)
		if err != nil {
			return nil, err
		}

		peers[i].insertGenesis()
	}

	return peers, nil
}

func TestAmIValidatorNow(t *testing.T) {
	peers, err := InitForTest(3, 3)
	if err != nil {
		t.Error(err)
	}

	for i := 0; i < 10; i++ {
		for _, peer := range peers {
			if peer.AmIValidatorNow() {
				block, err := peer.CreateBlock(peer.lastBlockNum+1, time.Now().Unix(), nil, peer.blocks[0].BlockHash)
				if err != nil {
					t.Error(err)
				}
				peers[0].insertBlock(block)
				peers[1].insertBlock(block)
				peers[2].insertBlock(block)
				break
			}
		}
	}

	require.Equal(t, 11, len(peers[0].blocks))
	require.Equal(t, 11, len(peers[1].blocks))
	require.Equal(t, 11, len(peers[2].blocks))
}

func TestSync(t *testing.T) {
		peers, err := InitForTest(3,3)
		if err != nil {
			t.Error(err)
		}

		for _, peer := range peers {
			if peer.AmIValidatorNow() {
				block, err := peer.CreateBlock(peer.lastBlockNum + 1, time.Now().Unix(), nil, peer.blocks[0].BlockHash)
				if err != nil {
					t.Error(err)
				}
				peer.insertBlock(block)
				break
			}
		}

		for i := 0; i < 3; i++ {
			for j := i + 1; j < 3; j++ {
				err := peers[i].AddPeer(peers[j])
				if err != nil {
					t.Error(err)
				}
			}
		}

		time.Sleep(time.Second)

		require.Equal(t, peers[1].blocks[1], peers[0].blocks[1])
		require.Equal(t, peers[2].blocks[1], peers[0].blocks[1])
}

func TestStartingBlockchain(t *testing.T) {
	peers, err := InitForTest(5, 3)
	if err != nil {
		t.Error(err)
	}

	for _, node := range peers {

		id := node.GetValidatorId()

		if id == 0 {
			transactions, err := node.PrepareTransactions()
			if err != nil {
				t.Error(err)
			}

			block, err := node.CreateBlock(node.lastBlockNum + 1, time.Now().Unix(), transactions, node.GetBlockByNumber(node.lastBlockNum).BlockHash)
			if err != nil {
				t.Error(err)
			}

			if err := node.insertBlock(block); err != nil {
				t.Error(err)
			}
			context := context.Background()
			message := Message{
				From: node.NodeAddress(),
				Data: block,
			}

			for _, peer := range node.peers {
				peer.Send(context, message)
			}
		}
	}

	tr := Transaction{
		From:   peers[3].NodeAddress(),
		To:     peers[4].NodeAddress(),
		Amount: 100,
		Fee:    10,
		PubKey: peers[3].NodeKey().(ed25519.PublicKey),
	}

	tr, err = peers[3].SignTransaction(tr)
	if err != nil {
		t.Fatal(err)
	}

	if err := peers[3].AddTransaction(tr); err != nil {
		t.Error(err)
	}

	context := context.Background()
	message := Message{
		From: peers[3].NodeAddress(),
		Data: tr,
	}

	peers[3].Broadcast(context, message)

	time.Sleep(time.Second * 5)
	for _, peer := range peers {
		peer.AmIValidatorNow()
	}
}

func TestSendTransactionSuccess(t *testing.T) {
	numOfPeers := 5
	numOfValidators := 3
	initialBalance := uint64(100000)
	peers := make([]Blockchain, numOfPeers)

	genesis := Genesis{
		make(map[string]uint64),
		make([]crypto.PublicKey, 0, numOfValidators),
	}

	keys := make([]ed25519.PrivateKey, numOfPeers)
	for i := range keys {
		_, key, err := ed25519.GenerateKey(nil)
		if err != nil {
			t.Fatal(err)
		}
		keys[i] = key
		if numOfValidators > 0 {
			genesis.Validators = append(genesis.Validators, key.Public())
			numOfValidators--
		} 

		address, err := PubKeyToAddress(key.Public())
		if err != nil {
			t.Error(err)
		}
		genesis.Alloc[address] = initialBalance
	}

	var err error
	for i := 0; i < numOfPeers; i++ {
		peers[i], err = NewNode(keys[i], genesis)
		if err != nil {
			t.Error(err)
		}
	}

	for i := 0; i < len(peers); i++ {
		for j := i + 1; j < len(peers); j++ {
			err = peers[i].AddPeer(peers[j])
			if err != nil {
				t.Error(err)
			}
		}
	}
	
	tr := Transaction{
		From:   peers[3].NodeAddress(),
		To:     peers[4].NodeAddress(),
		Amount: 100,
		Fee:    10,
		PubKey: keys[3].Public().(ed25519.PublicKey),
	}

	tr, err = peers[3].SignTransaction(tr)
	if err != nil {
		t.Fatal(err)
	}

	err = peers[0].AddTransaction(tr)
	if err != nil {
		t.Fatal(err)
	}

	//wait transaction processing
	time.Sleep(time.Second * 5)

	//check "from" balance
	balance, err := peers[0].GetBalance(peers[3].NodeAddress())
	if err != nil {
		t.Fatal(err)
	}

	if balance != initialBalance-100-10 {
		t.Fatal("Incorrect from balance")
	}

	//check "to" balance
	balance, err = peers[0].GetBalance(peers[4].NodeAddress())
	if err != nil {
		t.Fatal(err)
	}

	if balance != initialBalance+100 {
		t.Fatal("Incorrect to balance")
	}

	//check validators balance
	for i := 0; i < 3; i++ {
		balance, err = peers[0].GetBalance(peers[i].NodeAddress())
		if err != nil {
			t.Error(err)
		}

		if balance > initialBalance {
			t.Error("Incorrect validator balance")
		}
	}
}
