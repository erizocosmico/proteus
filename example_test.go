package proteus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gopkg.in/src-d/proteus.v1/example/client"
	"gopkg.in/src-d/proteus.v1/example/server"
)

func ExampleProteus() {
	addr := "localhost:8001"
	s, err := server.NewServer(addr)
	if err != nil {
		panic(fmt.Sprintf("could not open server: %s", err))
	}
	defer s.Stop()
	c, err := client.NewClient(addr)
	if err != nil {
		panic(fmt.Sprintf("could not open client: %s", err))
	}
	defer c.Close()

	n, err := c.RequestRandomNumber(0, 1)
	if err != nil {
		panic(fmt.Sprintf("could not receive random number: %s", err))
	}
	fmt.Println(n)

	cat, err := c.RequestRandomCategory()
	if err != nil {
		panic(fmt.Sprintf("could not receive category: %s", err))
	}
	fmt.Println(cat.CanBuy)
	fmt.Println(cat.ShowPrices)

	α, err := c.RequestAlphaTime()
	if err != nil {
		panic(fmt.Sprintf("could not receive alpha time: %s", err))
	}
	fmt.Printf("%s: %s\n", α.Name, α.Time.Format("Jan 2, 2006 at 3:04pm"))

	ω, err := c.RequestOmegaTime()
	if err != nil {
		panic(fmt.Sprintf("could not receive omega time: %s", err))
	}
	fmt.Printf("%s: %s\n", ω.Name, ω.Time.Format("Jan 2, 2006 at 3:04pm"))

	product, err := c.RequestPhone()
	if err != nil {
		panic(fmt.Sprintf("could not receive product: %s", err))
	}
	fmt.Println(product.Name)
	fmt.Println(strings.Join(product.Tags, ", "))

	duration, err := c.RequestDurationForLength(299792458)
	if err != nil {
		panic(fmt.Sprintf("could not receive duration: %s", err))
	}
	fmt.Printf("Duration: %d\n", duration.Duration/time.Second)

	go func() {
		err := server.RunGRPCGatewayServer(":8585", "localhost:8001")
		if err != nil {
			panic(fmt.Sprintf("error running GRPC gateway server: %s", err))
		}
	}()

	resp, err := http.Get("http://localhost:8585/sum/5/6")
	if err != nil {
		panic(err)
	}

	var result = make(map[string]interface{})
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		panic(err)
	}

	fmt.Printf("Sum 5 + 6: %v", result["result1"])

	// Output: 4
	// true
	// true
	// alpha: Jan 1, 1970 at 12:00am
	// omega: Dec 12, 2012 at 10:30am
	// MiPhone
	// cool, mi, phone
	// Duration: 1
	// Sum 5 + 6: 11
}
