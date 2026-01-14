package main

import (
	"fmt"
	"os"

	"github.com/frida/frida-go/frida"
	"github.com/gofiber/fiber/v2"
)

var script = `
        var modules = Process.enumerateModules();
                for (var i = 0; i < modules.length; i++) {
                    console.log(modules[i].name)
                }

`

func main() {
	mgr := frida.NewDeviceManager()

	devices, err := mgr.EnumerateDevices()
	if err != nil {
		panic(err)
	}

	for _, d := range devices {
		fmt.Println("[*] Found device with id:", d.ID())
	}

	localDev, err := mgr.LocalDevice()
	if err != nil {
		fmt.Println("Could not get local device: ", err)
		// Let's exit here because there is no point to do anything with nonexistent device
		os.Exit(1)
	}

	fmt.Println("[*] Chosen device: ", localDev.Name())

	fmt.Println("[*] Attaching to Telegram")
	session, err := localDev.Attach("Telegram.exe", nil)
	if err != nil {
		fmt.Println("Error occurred attaching:", err)
		os.Exit(1)
	}

	script, err := session.CreateScript(script)
	if err != nil {
		fmt.Println("Error occurred creating script:", err)
		os.Exit(1)
	}

	script.On("message", func(msg string) {
		fmt.Println("[*] Received", msg)
	})

	if err := script.Load(); err != nil {
		fmt.Println("Error loading script:", err)
		os.Exit(1)
	}

	// r := bufio.NewReader(os.Stdin)
	// r.ReadLine()

	//------I've just added some of the code below !!!   
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, Fiber!")
	})
	// app.Listen(":3000")
}
