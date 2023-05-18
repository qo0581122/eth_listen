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
					log.Fatal(err)
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
					if !IsWantToAddress(tx.To().String()) {
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
						SendDingRobot(dingUrl, fmt.Sprintf("[账户]:%s\n[交易地址]:%s\n[备注]:执行了其他操作\n", ad.String(), tx.Hash().Hex()), true)
						continue
					}
					target = "0x" + target[len(target)-40:] //土狗币
					fmt.Println(target)
					fbalance := new(big.Float)
					fbalance.SetString(tx.Value().String())
					ethValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(18)))
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

// 填入目标地址
func IsWantToAddress(addr string) bool {
	wants := []string{"0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D"}
	for _, v := range wants {
		if v == addr {
			return true
		}
	}
	return false
}

// 填入想要跟踪的用户地址
func IsWantFromAddress(addr string) bool {
	wants := []string{"0xAf2358e98683265cBd3a48509123d390dDf54534"}
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
