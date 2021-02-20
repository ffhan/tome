# Toy Order Matching Engine - TOME

TOME matches incoming buy/sell orders to create trades. It follows usual financial market order types, parameters and
market behaviour.

## CLI

As an example of the matching algorithm I've implemented a CLI that operates the order book.

`go run ./examples/cli/` for an interactive playground (enter your orders)
, `cat examples/cli/example.txt | go run ./examples/cli` for a predetermined example.

Instructions follow the following expression syntax:

* buy or sell
    * `<buy/sell> <number of shares> <market/limit> [if limit enter the limit price] [parameters, if stop then next parameter has to be the stop price, if GTD next param has to be the date]`
* print settings - `settings`
* print books - `print`
* change settings - `set <setting> [subsetting...] <yes/no/y/n/true/false/t/f>|value`
    * update setting according to its rules, currently supported are:
        1. `set print <always/never/trade>` - always print books, never print books or print books only when a trade
           occurs
        1. `set clear <yes/no/y/n/true/false/t/f>` - clear output on every instructio
        1. `set prompt <yes/no/y/n/true/false/t/f>` - prompt "enter instruction:" before each instruction
        1. `set print instructions <yes/no/y/n/true/false/t/f>` - print parsed instructions after each instruction
        1. `set print comments <yes/no/y/n/true/false/t/f>` - print comments?

Parameters are case insensitive. Comments start with `#` and are supported.

Examples:

* `buy 20 market gtc stop 25 ioc` - buy 20 shares at market price, parameters are GTC (good till cancelled), stop price
  at 25 and IOC (immediately or cancel)
* `sell 50 limit 24`  - sell 50 shares at limit price 24
* `buy 40 market`  - buy 40 shares at market price
* `sell 20 limit 23.56 stop 24 GFD` - sell 20 shares at limit price 23.56, set stop price at 24 + GTD (good for the day)
* `buy 10 limit 26 FOK` - buy 10 shares at limit 26 + FOK (fill or kill)
* `settings` - print out current settings
* `print` - print out the current state of the books

An example is provided at the end of the document and in `examples/cli/example.txt`.

## Currently supported

* order types
    * market order - execute an order as fast as possible, cross the spread
    * limit order - execute an order with a limit on bid/ask price (e.g. $x or less for a bid, or $y or more for an ask)
* order params
    * STOP - stop order, set a stop price which will activate the order once the market price crosses it
    * AON - all or nothing, don't allow partial fills
    * IOC - immediate or cancel, immediately fill what's possible, cancel the rest
    * FOK - AON+IOC, immediately match an order in full (without partial fills) or cancel it

## TODO

* [x] stop orders
* [ ] GFD, GTC, GTD parameters
* [ ] logic surrounding the order book - trading hours, pre/after market restrictions
* [ ] basic middle & back office functionalities - risk assessment, limits
* [ ] TCP/UDP server that accepts orders
* [ ] reporting market volume, share price
* [ ] reporting acknowledgments & updates to clients (share price, displayed/hidden orders...)

## Market behaviour

Market orders are always given priority above all other orders, then sorted according to time of arrival.

* orders are FIFO based
    * bids - price (descending), time (ascending)
    * asks - price (ascending), time (ascending)
    * quantity does not matter in sorting
* market price is set at the last trade price
* stop bids are activated once the market price is above or equal the stop price
* stop asks are activated once the market price is below or equal the stop price

When a match occurs between two limit orders the price is set on the bid price. Bid of $25 and ask of $24 will be
matched at $25.

## Architecture (in development)

Order book & trade books are per-instrument objects, one order book can only handle one instrument.

* order book - stores active orders in memory, handles order matching
* trade book - stores daily trades in memory, provides additional data about trading
* order container - container for efficient order insertion, search, traversal and removal
* order repository - persistent storage of orders
* trade repository - persistent storage of trades

### Order book

* order repository is used to persist all orders
* it uses two treemap data structures for ask and bid orders
    * key is an OrderTracker object which contains necessary info to track an order and sort it
* active orders are stored in a hashmap for fast lookup (by order ID) and storage
* order trackers are stored in a hashmap - used to lookup order trackers (usually to be able to search a treemap)

## Performance

The current figures are without implemented stop orders, manual cancellations and persistent storage. They are mostly a
representation of in-memory order matching throughput.

`BenchmarkOrderBook_Add-12    	  813372	      1764 ns/op	     663 B/op	       8 allocs/op`

Each order insertion to the order book takes about 1.8 μs, which means we can (theoretically) match ~560k orders in 1
second.

After all insertions 568102904 bytes (~568MB in use for about 460k active orders - ~1.2kB/order) are in use, before
insertions 210422952 bytes. Reported allocations are around 663 B/op. About 62% of total memory usage comes through the
Add method, 164% increase from the setup state.

A big performance hit was suffered after stop orders were implemented - further optimizations will be necessary.
Memory ballast definitely improves performance (about 3% improvement, which means additional 53k orders per second), but
guarantees the process a certain amount of memory - which is not free to use by the rest of the system.

Huge performance improvements came from OrderTracker tracking only nanoseconds as timestamps, prices as float64s and
smarter memory management. My goal is to hit 500 ns/op to be able to hit 2 million operations per second.

Current benchmark figures are currently without stop orders, but with a *huge* backlog of 400k unfilled orders, which definitely
isn't a realistic scenario. Further improvements to randomization of orders will be necessary.

`cockroachdb/apd` gave a significant performance improvement over `shopspring/decimal` mainly because of huge memory
usage improvement which drastically lowered the allocation rates (from 44 alloc/op to 4 alloc/op).

## CLI example

Example entries and explanation can be found at `./examples/cli/example.txt`

```bash
> cat examples/cli/example.txt | go run ./examples/cli

instructions: [set clear false]
enter instruction:instructions: [set prompt no ]
instructions: [set print instructions n ]
# buy 200 shares at market price
# following stop bids will be activated when market price reaches above their stop prices,
# but the order price is not the same as stop price
# set a stop bid at limit 25, activated when market price passes 24
# set a stop bid at limit 24, activated when market price passes 23
+----+--------+--------+------------+--------------------------------+-----+-----------+--------+
| ID |  TYPE  | PRICE  | STOP PRICE |              TIME              | QTY | FILLEDQTY | PARAMS |
+----+--------+--------+------------+--------------------------------+-----+-----------+--------+
|  1 | Market | 0.0000 |     0.0000 | 2021-02-20 14:00:12.438923503  | 200 |         0 |        |
|    |        |        |            | +0100 CET m=+0.001610273       |     |           |        |
+----+--------+--------+------------+--------------------------------+-----+-----------+--------+
bids
+----+------+-------+------------+------+-----+-----------+--------+
| ID | TYPE | PRICE | STOP PRICE | TIME | QTY | FILLEDQTY | PARAMS |
+----+------+-------+------------+------+-----+-----------+--------+
+----+------+-------+------------+------+-----+-----------+--------+
asks
+----+-------+---------+------------+--------------------------------+-----+-----------+--------+
| ID | TYPE  |  PRICE  | STOP PRICE |              TIME              | QTY | FILLEDQTY | PARAMS |
+----+-------+---------+------------+--------------------------------+-----+-----------+--------+
|  3 | Limit | 24.0000 |    23.0000 | 2021-02-20 14:00:12.438945514  |  30 |         0 | STOP   |
|    |       |         |            | +0100 CET m=+0.001632269       |     |           |        |
|  2 | Limit | 25.0000 |    24.0000 | 2021-02-20 14:00:12.438936978  |  20 |         0 | STOP   |
|    |       |         |            | +0100 CET m=+0.001623736       |     |           |        |
+----+-------+---------+------------+--------------------------------+-----+-----------+--------+
stop bids
+----+------+-------+------------+------+-----+-----------+--------+
| ID | TYPE | PRICE | STOP PRICE | TIME | QTY | FILLEDQTY | PARAMS |
+----+------+-------+------------+------+-----+-----------+--------+
+----+------+-------+------------+------+-----+-----------+--------+
stop asks
+------+-------+-------+-----+-------+-------+
| TIME | BIDID | ASKID | QTY | PRICE | TOTAL |
+------+-------+-------+-----+-------+-------+
+------+-------+-------+-----+-------+-------+
trades
Market price: 20.25
# fill-or-kill sell 100 shares at limit of 23.5 (don't sell below that)
+----+--------+---------+------------+--------------------------------+-----+-----------+--------+
| ID |  TYPE  |  PRICE  | STOP PRICE |              TIME              | QTY | FILLEDQTY | PARAMS |
+----+--------+---------+------------+--------------------------------+-----+-----------+--------+
|  1 | Market |  0.0000 |     0.0000 | 2021-02-20 14:00:12.438923503  | 200 |       100 |        |
|    |        |         |            | +0100 CET m=+0.001610273       |     |           |        |
|  3 | Limit  | 24.0000 |    23.0000 | 2021-02-20 14:00:12.438945514  |  30 |         0 | STOP   |
|    |        |         |            | +0100 CET m=+0.001632269       |     |           |        |
+----+--------+---------+------------+--------------------------------+-----+-----------+--------+
bids
+----+------+-------+------------+------+-----+-----------+--------+
| ID | TYPE | PRICE | STOP PRICE | TIME | QTY | FILLEDQTY | PARAMS |
+----+------+-------+------------+------+-----+-----------+--------+
+----+------+-------+------------+------+-----+-----------+--------+
asks
+----+-------+---------+------------+--------------------------------+-----+-----------+--------+
| ID | TYPE  |  PRICE  | STOP PRICE |              TIME              | QTY | FILLEDQTY | PARAMS |
+----+-------+---------+------------+--------------------------------+-----+-----------+--------+
|  2 | Limit | 25.0000 |    24.0000 | 2021-02-20 14:00:12.438936978  |  20 |         0 | STOP   |
|    |       |         |            | +0100 CET m=+0.001623736       |     |           |        |
+----+-------+---------+------------+--------------------------------+-----+-----------+--------+
stop bids
+----+------+-------+------------+------+-----+-----------+--------+
| ID | TYPE | PRICE | STOP PRICE | TIME | QTY | FILLEDQTY | PARAMS |
+----+------+-------+------------+------+-----+-----------+--------+
+----+------+-------+------------+------+-----+-----------+--------+
stop asks
+--------------------------------+-------+-------+-----+---------+-------+
|              TIME              | BIDID | ASKID | QTY |  PRICE  | TOTAL |
+--------------------------------+-------+-------+-----+---------+-------+
| 2021-02-20 14:00:12.439841334  |     1 |     4 | 100 | 23.5000 |  2350 |
| +0100 CET m=+0.002528089       |       |       |     |         |       |
+--------------------------------+-------+-------+-----+---------+-------+
trades
Market price: 23.5000
# last order will be matched with the first order, market price is now set at 23.5
# market price 23.5 activates the stop order set at 23, it's added to the books but isn't matched since there aren't opposing sellers
# sell 150 shares at market price
+----+------+-------+------------+------+-----+-----------+--------+
| ID | TYPE | PRICE | STOP PRICE | TIME | QTY | FILLEDQTY | PARAMS |
+----+------+-------+------------+------+-----+-----------+--------+
+----+------+-------+------------+------+-----+-----------+--------+
bids
+----+------+-------+------------+------+-----+-----------+--------+
| ID | TYPE | PRICE | STOP PRICE | TIME | QTY | FILLEDQTY | PARAMS |
+----+------+-------+------------+------+-----+-----------+--------+
+----+------+-------+------------+------+-----+-----------+--------+
asks
+----+------+-------+------------+------+-----+-----------+--------+
| ID | TYPE | PRICE | STOP PRICE | TIME | QTY | FILLEDQTY | PARAMS |
+----+------+-------+------------+------+-----+-----------+--------+
+----+------+-------+------------+------+-----+-----------+--------+
stop bids
+----+------+-------+------------+------+-----+-----------+--------+
| ID | TYPE | PRICE | STOP PRICE | TIME | QTY | FILLEDQTY | PARAMS |
+----+------+-------+------------+------+-----+-----------+--------+
+----+------+-------+------------+------+-----+-----------+--------+
stop asks
+--------------------------------+-------+-------+-----+---------+-------+
|              TIME              | BIDID | ASKID | QTY |  PRICE  | TOTAL |
+--------------------------------+-------+-------+-----+---------+-------+
| 2021-02-20 14:00:12.439841334  |     1 |     4 | 100 | 23.5000 |  2350 |
| +0100 CET m=+0.002528089       |       |       |     |         |       |
| 2021-02-20 14:00:12.440757787  |     1 |     5 | 100 | 24.0000 |  2400 |
| +0100 CET m=+0.003444541       |       |       |     |         |       |
| 2021-02-20 14:00:12.440759688  |     2 |     5 |  20 | 25.0000 |   500 |
| +0100 CET m=+0.003446443       |       |       |     |         |       |
| 2021-02-20 14:00:12.440760289  |     3 |     5 |  30 | 24.0000 |   720 |
| +0100 CET m=+0.003447043       |       |       |     |         |       |
+--------------------------------+-------+-------+-----+---------+-------+
trades
Market price: 24.0000
# sell order is matched with the recently activated stop order and sold at price of 24
# the new market price is now 24, which activates the first stop order and is sold at price of 25
# the sell order has been matched first against the order 1 market bid, then the last stop bid at price 25 and
# then at the first stop bid at price 24
# stop orders have not been matched in order of time when they were added to the books, but ordered by price and then time

```

## Acknowledgements

* Practical .NET for Financial Markets by Samir Jayaswal and Yogesh Shetty
    * excellent reading material for functional and technical details about financial markets
    * good explanation of the order matching algorithm
* https://web.archive.org/web/20110219163448/http://howtohft.wordpress.com/2011/02/15/how-to-build-a-fast-limit-order-book/
    * insight into technical aspects regarding trading speed, efficiency...
* https://www.investopedia.com/investing/basics-trading-stock-know-your-orders/
    * great summary of order types
* https://github.com/enewhuis/liquibook
    * inspiration for some of the data structures and approaches to the problem
* https://github.com/google/uuid - a great UUID library
* https://github.com/igrmk/treemap - really easy to use treemap implementation, great code generation capabilities
* https://github.com/olekukonko/tablewriter - the best ASCII table writer in existence
* https://github.com/cockroachdb/apd - fast & relatively memory efficient decimal Go implementation
