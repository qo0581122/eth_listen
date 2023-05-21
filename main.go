package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	dingUrl = "https://oapi.dingtalk.com/robot/send?access_token=72dfa9c68ec2f5da80d86e60362e6fc3bb3045294c228254dc00740f1f6fa5e7"
)

func main() {
	ctx := context.Background()
	client, err := ethclient.DialContext(ctx, "wss://mainnet.infura.io/ws/v3/3bf2cf0bb9e840e89f471b6359a2add0")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer client.Close()

	headers := make(chan *types.Header, 10)

	sub, err := client.SubscribeNewHead(ctx, headers)
	if err != nil {
		log.Fatal(err)
	}
	chainID, err := client.NetworkID(ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("server start now!")

	go func() {
		for {
			select {
			case err := <-sub.Err():
				log.Fatal(err)
			case head := <-headers:
				log.Println("accept block, block number:", head.Number.String())
				block, err := client.BlockByNumber(ctx, head.Number)
				if err != nil {
					log.Println(err)
					continue
				}
				if block == nil {
					log.Println("block == nil")
					continue
				}
				log.Println("block info, tx count:", len(block.Transactions()))
				for _, tx := range block.Transactions() {
					ad, err := types.Sender(types.NewLondonSigner(chainID), tx)
					if err != nil {
						continue
					}
					if tx.To() == nil {
						continue
					}
					fbalance := new(big.Float)
					fbalance.SetString(tx.Value().String())
					ethValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(18)))
					if d, _ := ethValue.Int64(); d < 1 {
						continue
					}
					if !IsWantFromAddress(ad.String()) {
						continue
					}
					bys, _ := tx.MarshalJSON()
					m := make(map[string]interface{})
					_ = json.Unmarshal(bys, &m)
					target := m["input"].(string)
					if len(target) <= 40 {
						SendDingRobot(dingUrl, fmt.Sprintf("[账户]:%s\n[交易地址]:%s\n[价值]:%v\n[备注]:执行了其他操作\n", ad.String(), tx.Hash().Hex(), ethValue), true)
						continue
					}
					target = "0x" + target[len(target)-40:] //土狗币
					SendDingRobot(dingUrl, fmt.Sprintf("[交易地址]:%s\n[账户]:%s\n[购入]:%s\n[价值]:%v\n", tx.Hash().Hex(), ad.String(), target, ethValue), true)
				}
			}
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		s := <-sigChan
		log.Printf("get a signal %s\n", s.String())
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			log.Println("server exit now...")
			return
		case syscall.SIGHUP:
		default:
		}
	}
}

// 填入想要跟踪的用户地址
func IsWantFromAddress(addr string) bool {
	wants := []string{
		"0xAf2358e98683265cBd3a48509123d390dDf54534",
		"0x9cd6140c2de8af7595629bcca099497f0c28b2a9",
		"0x25cd302e37a69d70a6ef645daea5a7de38c66e2a",
		"0x4a2c786651229175407d3a2d405d1998bcf40614",
		"0x5bdf85216ec1e38d6458c870992a69e38e03f7ef",
		"0xaf2358e98683265cbd3a48509123d390ddf54534",
		"0xc758d5718147c8c5bc440098d623cf8d96b95b83",
		"0x3c5883c650d600bd543a9b5c8d9a3a6f5d16b8f4",
		"0x74de5d4fcbf63e00296fd95d33236b9794016631",
		"0x0d50c708780432e0301c8e07ee26b891904e211a",
		"0x94d358f88dde695a30578ee7584e7cdda901e58f",
		"0xb090fbbb0dccacb8bd91f7a5263321ec660fe248",
		"0x601a63c50448477310fedb826ed0295499baf623",
		"0x8798b90c5192a01f9981e53dc5c9d8fd50108d20",
		"0x1991b6a81324a6574933d133666742e8643283cd",
		"0x64e518286dd24de1410b16f33e74fd0c88022162",
		"0xa6b86cff41251cc85a33f784c9ea61cf011372d1",
		"0xbf107cd8d91e983690a34c565b1ad978a5b6729f",
		"0x8999020b433a0ac8efdbfe49ef03855aef262eba",
		"0xf5ead33d22c0abe56b00b341d7a0023a77bd5ee4",
		"0x00000000009726632680fb29d3f7a9734e3010e2",
		"0x4c90cdc854af7ac9fbc522c08fb4f4116c61bb0f",
		"0x601a63c50448477310fedb826ed0295499baf623",
		"0x6a75774982e73a4444d1364ed894ac58412b8c56",
	}
	for _, v := range wants {
		if v == addr {
			return true
		}
	}
	return false
}

func SendDingRobot(url string, content string, isAtAll bool) error {
	msg := dingtextMessage{MsgType: "text", Text: dingtextParams{Content: content}, At: dingdingAt{IsAtAll: isAtAll}}
	return doSendDingRobot(url, msg)
}

func doSendDingRobot(url string, msg interface{}) error {
	m, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(m))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var dr dingResponse
	err = json.Unmarshal(data, &dr)
	if err != nil {
		return err
	}
	if dr.Errcode != 0 {
		return fmt.Errorf("dingrobot send failed: %v", dr.Errmsg)
	}
	return nil
}

type dingResponse struct {
	Errcode int
	Errmsg  string
}
type dingtextMessage struct {
	MsgType string         `json:"msgtype"`
	Text    dingtextParams `json:"text"`
	At      dingdingAt     `json:"at"`
}

type dingtextParams struct {
	Content string `json:"content"`
}

// At at struct
type dingdingAt struct {
	AtMobiles []string `json:"atMobiles"`
	IsAtAll   bool     `json:"isAtAll"`
}
