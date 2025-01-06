package main

import (
	"C"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
)

const PluginName = "telegram"
const PlugingDesc = "Telegram"

type TelegramCfg struct {
	chatId string
	apiKey string
}

//export FLBPluginRegister
func FLBPluginRegister(def unsafe.Pointer) int {
	// Gets called only once when the plugin.so is loaded
	log.Printf("[%s] registering plugin", PluginName)
	return output.FLBPluginRegister(def, PluginName, PlugingDesc)
}

//export FLBPluginInit
func FLBPluginInit(plugin unsafe.Pointer) int {
	// Gets called only once for each instance you have configured.
	apiKey := output.FLBPluginConfigKey(plugin, "api_key")
	chatId := output.FLBPluginConfigKey(plugin, "chat_id")

	log.Printf("[%s] [info] api_key = %q, chat_id = %q", PluginName, apiKey, chatId)

	// Set the context to point to any Go variable
	output.FLBPluginSetContext(plugin, TelegramCfg{
		apiKey: apiKey,
		chatId: chatId,
	})

	return output.FLB_OK
}

//export FLBPluginFlushCtx
func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, tag *C.char) int {
	// Gets called with a batch of records to be written to an instance.

	// Type assert context back into the original type for the Go variable
	cfg := output.FLBPluginGetContext(ctx).(TelegramCfg)

	dec := output.NewDecoder(data, int(length))

	for {
		ret, ts, record := output.GetRecord(dec)
		if ret != 0 {
			break
		}

		// Print record keys and values
		timestamp := ts.(output.FLBTime)
		str := fmt.Sprintf("%s %s\n", C.GoString(tag), timestamp.String())

		for k, v := range record {
			str += fmt.Sprintf("%s: %s\n", k, v)
		}

		if err := SendTelegramMessage(cfg.apiKey, cfg.chatId, str); err != nil {
			log.Printf("[%s] [error] telegram notification failed: %+v", PluginName, err)
			return output.FLB_ERROR
		}
	}

	return output.FLB_OK
}

//export FLBPluginExit
func FLBPluginExit() int {
	log.Printf("[%s] [info] exit", PluginName)
	return output.FLB_OK
}

func SendTelegramMessage(apiKey, chatId, text string) error {
	data := url.Values{}
	data.Set("chat_id", chatId)
	data.Set("text", text)

	resp, err := http.Post(fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", apiKey), "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("telegram api return error status code: %d", resp.StatusCode)
	}

	return nil
}

func main() {
}
