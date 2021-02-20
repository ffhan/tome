package main

import (
	"bufio"
	"fmt"
	"github.com/cockroachdb/apd"
	"github.com/ffhan/tome"
	"github.com/google/uuid"
	"github.com/olekukonko/tablewriter"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var currentOrderID uint64

type printEvent byte

const (
	printNever  printEvent = iota
	printAlways printEvent = iota
	printOnTrade
)

type settings struct {
	printEvent        printEvent
	clearBeforePrint  bool
	printComments     bool
	printInstructions bool
	instructionPrompt bool
}

func main() {
	const instrument = "TEST"
	tb := tome.NewTradeBook(instrument)
	ob := tome.NewOrderBook(instrument, *apd.New(2025, -2), tb, tome.NOPOrderRepository)

	s := settings{
		printEvent:        printAlways,
		clearBeforePrint:  true,
		printComments:     false,
		instructionPrompt: true,
		printInstructions: true,
	}

	scanner := bufio.NewScanner(os.Stdin)

	lenTrades := 0

	for {
		if s.instructionPrompt {
			fmt.Print("enter instruction:")
		}
		if !scanner.Scan() {
			break
		}

		if s.clearBeforePrint {
			c := exec.Command("clear")
			c.Stdout = os.Stdout
			c.Run()
		}

		text := scanner.Text()
		if strings.HasPrefix(text, "#") { // if text begins with a comment ignore the line
			if s.printComments {
				fmt.Println(text)
			}
			continue
		}
		commentIndex := strings.Index(text, "#")
		if commentIndex == 0 { // whole line is a comment, skip the line
			continue
		} else if commentIndex > 0 {
			comment := text[commentIndex:]
			text = text[:commentIndex]
			if s.printComments {
				fmt.Println(comment)
			}
		}
		split := strings.Split(text, " ")

		if s.printInstructions {
			fmt.Printf("instructions: %v\n", split)
		}
		action := split[0]

		switch action {
		case "print":
			print(ob, tb)
			continue
		case "buy":
			order(tome.SideBuy, ob, split)
		case "sell":
			order(tome.SideSell, ob, split)
		case "set":
			updateSettings(&s, split)
			continue
		case "settings":
			fmt.Printf("%+v\n", s)
		}

		switch s.printEvent {
		case printAlways:
			print(ob, tb)
		case printOnTrade:
			newTrades := len(tb.DailyTrades())
			if newTrades > lenTrades {
				lenTrades = newTrades
				print(ob, tb)
			}
		}
	}
}

func updateSettings(s *settings, split []string) {
	switch split[1] {
	case "print":
		switch split[2] {
		case "always":
			s.printEvent = printAlways
		case "never":
			s.printEvent = printNever
		case "trade":
			s.printEvent = printOnTrade
		case "comments":
			switch split[3] {
			case "true", "y", "yes", "t":
				s.printComments = true
			case "false", "n", "no", "f":
				s.printComments = false
			default:
				log.Println("invalid print comment setting")
			}
		case "instructions":
			switch split[3] {
			case "true", "y", "yes", "t":
				s.printInstructions = true
			case "false", "n", "no", "f":
				s.printInstructions = false
			default:
				log.Println("invalid print instructions setting")
			}
		default:
			log.Println("invalid print setting")
		}
	case "clear":
		switch split[2] {
		case "true", "y", "yes", "t":
			s.clearBeforePrint = true
		case "false", "n", "no", "f":
			s.clearBeforePrint = false
		default:
			log.Println("invalid clear setting")
		}
	case "prompt":
		switch split[2] {
		case "true", "y", "yes", "t":
			s.instructionPrompt = true
		case "false", "n", "no", "f":
			s.instructionPrompt = false
		}
	}
}

func order(side tome.OrderSide, ob *tome.OrderBook, split []string) {
	const (
		orderQty = iota + 1
		orderType
		orderPrice
		orderParams
	)
	currentOrderID += 1

	var price, stopPrice float64
	var Type tome.OrderType
	var err error
	if split[orderType] == "market" {
		price = 0
		Type = tome.TypeMarket
	} else if split[orderType] == "limit" {
		price, err = strconv.ParseFloat(split[orderPrice], 64)
		if err != nil {
			panic(err)
		}
		Type = tome.TypeLimit
	} else {
		log.Println("invalid order type")
		return
	}

	qty, err := strconv.Atoi(split[orderQty])
	if err != nil {
		panic(err)
	}

	var params tome.OrderParams

	oParams := orderParams
	if Type == tome.TypeMarket {
		oParams -= 1
	}

	for i, param := range split[oParams:] { // todo: after GFD & STOP expect a value
		switch param {
		case "aon":
			params |= tome.ParamAON
		case "stop":
			params |= tome.ParamStop
			stopPrice, err = strconv.ParseFloat(split[oParams+i+1], 64)
			if err != nil {
				panic(err)
			}
			i += 1
		case "ioc":
			params |= tome.ParamIOC
		case "fok":
			params |= tome.ParamFOK
		case "gtc":
			params |= tome.ParamGTC
		case "gfd":
			params |= tome.ParamGFD
		case "gtd":
			params |= tome.ParamGTD
			//time.Parse(time.RFC822Z, split[len(split)-1])
		}
	}

	order := tome.Order{
		ID:         currentOrderID,
		Instrument: "TEST",
		CustomerID: uuid.UUID{},
		Timestamp:  time.Now(),
		Type:       Type,
		Params:     params,
		Qty:        int64(qty),
		FilledQty:  0,
		Price:      *apd.New(int64(price*10000), -4),
		StopPrice:  *apd.New(int64(stopPrice*10000), -4),
		Side:       side,
		Cancelled:  false,
	}
	if _, err := ob.Add(order); err != nil {
		panic(err)
	}
}

func print(ob *tome.OrderBook, tb *tome.TradeBook) {
	bids := ob.GetBids()
	asks := ob.GetAsks()

	stopBids := ob.GetStopBids()
	stopAsks := ob.GetStopAsks()

	printOrders("bids", bids)
	printOrders("asks", asks)
	printOrders("stop bids", stopBids)
	printOrders("stop asks", stopAsks)
	trades := tb.DailyTrades()
	printTrades(trades)
	marketPrice := ob.MarketPrice()
	fmt.Printf("Market price: %s\n", marketPrice.String())
}

func printTrades(trades []tome.Trade) {
	writer := tablewriter.NewWriter(os.Stdout)
	writer.SetHeader([]string{"time", "BidID", "AskID", "qty", "price", "total"})
	for _, trade := range trades {

		price, _ := trade.Price.Float64()
		qty := trade.Qty

		writer.Append([]string{trade.Timestamp.String(), strconv.Itoa(int(trade.BidOrderID)), strconv.Itoa(int(trade.AskOrderID)),
			strconv.Itoa(int(trade.Qty)), trade.Price.String(), strconv.FormatFloat(price*float64(qty), 'f', -1, 64)})
	}
	writer.SetCaption(true, "trades")
	writer.Render()
}

func printOrders(title string, orders []tome.Order) {
	writer := tablewriter.NewWriter(os.Stdout)
	writer.SetHeader([]string{"ID", "type", "price", "stop price", "time", "qty", "filledQty", "params"})
	for _, order := range orders {
		writer.Append([]string{strconv.Itoa(int(order.ID)), order.Type.String(), order.Price.String(), order.StopPrice.String(),
			order.Timestamp.String(), strconv.Itoa(int(order.Qty)), strconv.Itoa(int(order.FilledQty)), order.Params.String()})
	}
	writer.SetCaption(true, title)
	writer.Render()
}
