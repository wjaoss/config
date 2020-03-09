package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/wjaoss/config"
	"github.com/wjaoss/x/lib/security"
)

var (
	configKey = "4755E2EDAB241FB28BC69E34783D2758"
)

func encodeFile(src, dest string) error {
	_, err := os.Stat(src)
	if os.IsNotExist(err) {
		return errors.New("Configuration file not found")
	}

	// Read encrypted file content
	// #nosec
	plainText, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	key, err := hex.DecodeString(configKey)
	if err != nil {
		return err
	}

	// Encrypt the content
	cipherText, err := security.Encrypt(key, plainText)
	if err != nil {
		return err
	}

	encoded := []byte(hex.EncodeToString(cipherText))

	// Write to new file
	return ioutil.WriteFile(dest, encoded, os.ModePerm)
}

func decode(src []byte) []byte {
	cipherText, err := hex.DecodeString(string(src))
	if err != nil {
		return src
	}

	// Decode the encryption key
	key, err := hex.DecodeString(configKey)
	if err != nil {
		return src
	}

	plainText, err := security.Decrypt(key, cipherText)
	if err != nil {
		return src
	}

	return plainText
}

type decoder struct{}

func (d *decoder) Decode(src []byte) []byte {
	return decode(src)
}

func main() {
	var name, slogan, nested1 string
	var nested2 int

	flag.StringVar(&name, "name", "", "person name")
	flag.StringVar(&slogan, "slogan", "", "person slogan")
	flag.StringVar(&nested1, "nested.props.deep", "", "cli nested")
	flag.IntVar(&nested2, "nested.props.really", 55, "cli nested")

	flag.Parse()

	// // TODO: create runner here which watching remote config. for every changeset, dump its value
	// for testing, encode sample file
	if err := encodeFile("test.json", "test-encoded.json"); err != nil {
		log.Fatal(err)
	}

	// <- s.Notify()
	ctx, cancel := context.WithCancel(context.Background())

	c := config.New(
		config.WithSource(
			config.File("test-encoded.json"),
			&decoder{},
		),
		config.WithSource(
			config.File("test.yaml", true),
		),
		config.WithSource(
			config.Cli(),
		),
		config.WithSource(
			config.Etcd("127.0.0.1:2379", config.EtcdOption{
				Prefix:      "/configuration/app",
				DialTimeout: time.Second * 2,
			}),
		),
		config.EnableWatcher(ctx, time.Second*5),
	)

	fmt.Println(string(c.Bytes()))
	fmt.Println("alert?", string(c.Get("alert.enabled").Bytes()))

	go func() {
		changed := c.Subscribe()

		for {
			<-changed
			fmt.Println("updated")

			fmt.Println(string(c.Bytes()))

			fmt.Println("alert?", c.Get("alert.enabled").Bool(false))
		}
	}()

	fmt.Scanln()
	fmt.Println("canceling")

	cancel()
}
