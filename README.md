# Toy Order Matching Engine - TOME

TOME matches incoming buy/sell orders to create trades. It follows usual financial market order types, parameters and
market behaviour.

## CLI

As an example of the matching algorithm I've implemented a CLI that operates the order book.

`go run ./examples/cli/` for an interactive playground (enter your orders)
, `cat examples/cli/example.txt | go run ./examples/cli` for a predetermined example.

Adding order follows the following expression syntax:
`<buy/sell> <number of shares> <market/limit> [if limit enter the limit price] [parameters, if stop then next parameter has to be the stop price, if GTD next param has to be the date]`

Parameters are case insensitive.

Examples:
* `buy 20 market gtc stop 25 ioc` - buy 20 shares at market price, parameters are GTC (good till cancelled), stop price at 25 and IOC (immediately or cancel)
* `sell 50 limit 24`  - sell 50 shares at limit price 24
* `buy 40 market`  - buy 40 shares at market price
* `sell 20 limit 23.56 stop 24 GFD` - sell 20 shares at limit price 23.56, set stop price at 24 + GTD (good for the day) 
* `buy 10 limit 26 FOK` - buy 10 shares at limit 26 + FOK (fill or kill)

An example is provided at the end of the document.

## Currently supported

* order types
    * market order - execute an order as fast as possible, cross the spread
    * limit order - execute an order with a limit on bid/ask price (e.g. $x or less for a bid, or $y or more for an ask)
* order params
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

`BenchmarkOrderBook_Add-12    	  707324	      1977 ns/op	     646 B/op	       8 allocs/op`

Each order insertion to the order book takes about 2 μs, which means we can (theoretically) match ~500k orders in 1
second.

After all insertions 454418808 bytes (~454B in use for about 380k active orders - ~1.2kB/order) are in use, before
insertions 147276648 bytes. Reported allocations are around 646 B/op. About 68% of total memory usage comes through the
Add method, 209% increase from the setup state.

A big performance hit was suffered after stop orders were implemented - further optimizations will be necessary.

Huge performance improvements came from OrderTracker tracking only nanoseconds as timestamps, prices as float64s and
smarter memory management. My goal is to hit 500 ns/op to be able to hit 2 million operations per second.

`cockroachdb/apd` gave a significant performance improvement over `shopspring/decimal` mainly because of huge memory
usage improvement which drastically lowered the allocation rates (from 44 alloc/op to 4 alloc/op).

## CLI example

1. buy 20 shares, market price at stop price of 25
1. sell 50 shares, limit price of 24
    * stop bid from point 1. is not filled because current market price is 20.25
1. buy 40 shares, market price
    * ask from point 2 is matched with this order, creating a trade of 40 shares at price 24
    * market price is now 24, so order 1 is still not activated (since market price is lower than stop price of 25)
1. sell 20 shares, limit price 23.56
    * no opposing bids, so no matches (stop order 1 is still not activated)
1. buy 10 shares, limit price of 26 - fill or kill (fill the whole order or cancel it immediately)
    * matches with order 4 at price 26 for 10 shares - market price is now 26
    * new market price 26 activates the stop order 1, adding it to the books
        * there are two asks: 10 shares at 23.56 and 10 shares at 24 (sorted by precedence)
    * newly activated stop order (which was a market order) matches with two asks
    * stop order of 20 shares is filled in two trades: 10 at 23.56 and 10 at 24

Example:

```
cat examples/cli/example.txt | go run ./examples/cli
enter instruction:instructions: [buy 20 market stop 25]
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
+----+--------+--------+------------+--------------------------------+-----+-----------+--------+
| ID |  TYPE  | PRICE  | STOP PRICE |              TIME              | QTY | FILLEDQTY | PARAMS |
+----+--------+--------+------------+--------------------------------+-----+-----------+--------+
|  1 | Market | 0.0000 |    25.0000 | 2021-02-20 01:15:03.837669981  |  20 |         0 | STOP   |
|    |        |        |            | +0100 CET m=+0.001059140       |     |           |        |
+----+--------+--------+------------+--------------------------------+-----+-----------+--------+
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
enter instruction:instructions: [sell 50 limit 24]
+----+------+-------+------------+------+-----+-----------+--------+
| ID | TYPE | PRICE | STOP PRICE | TIME | QTY | FILLEDQTY | PARAMS |
+----+------+-------+------------+------+-----+-----------+--------+
+----+------+-------+------------+------+-----+-----------+--------+
bids
+----+-------+---------+------------+--------------------------------+-----+-----------+--------+
| ID | TYPE  |  PRICE  | STOP PRICE |              TIME              | QTY | FILLEDQTY | PARAMS |
+----+-------+---------+------------+--------------------------------+-----+-----------+--------+
|  2 | Limit | 24.0000 |     0.0000 | 2021-02-20 01:15:03.838227556  |  50 |         0 |        |
|    |       |         |            | +0100 CET m=+0.001616705       |     |           |        |
+----+-------+---------+------------+--------------------------------+-----+-----------+--------+
asks
+----+--------+--------+------------+--------------------------------+-----+-----------+--------+
| ID |  TYPE  | PRICE  | STOP PRICE |              TIME              | QTY | FILLEDQTY | PARAMS |
+----+--------+--------+------------+--------------------------------+-----+-----------+--------+
|  1 | Market | 0.0000 |    25.0000 | 2021-02-20 01:15:03.837669981  |  20 |         0 | STOP   |
|    |        |        |            | +0100 CET m=+0.001059140       |     |           |        |
+----+--------+--------+------------+--------------------------------+-----+-----------+--------+
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
enter instruction:instructions: [buy 40 market]
+----+------+-------+------------+------+-----+-----------+--------+
| ID | TYPE | PRICE | STOP PRICE | TIME | QTY | FILLEDQTY | PARAMS |
+----+------+-------+------------+------+-----+-----------+--------+
+----+------+-------+------------+------+-----+-----------+--------+
bids
+----+-------+---------+------------+--------------------------------+-----+-----------+--------+
| ID | TYPE  |  PRICE  | STOP PRICE |              TIME              | QTY | FILLEDQTY | PARAMS |
+----+-------+---------+------------+--------------------------------+-----+-----------+--------+
|  2 | Limit | 24.0000 |     0.0000 | 2021-02-20 01:15:03.838227556  |  50 |        40 |        |
|    |       |         |            | +0100 CET m=+0.001616705       |     |           |        |
+----+-------+---------+------------+--------------------------------+-----+-----------+--------+
asks
+----+--------+--------+------------+--------------------------------+-----+-----------+--------+
| ID |  TYPE  | PRICE  | STOP PRICE |              TIME              | QTY | FILLEDQTY | PARAMS |
+----+--------+--------+------------+--------------------------------+-----+-----------+--------+
|  1 | Market | 0.0000 |    25.0000 | 2021-02-20 01:15:03.837669981  |  20 |         0 | STOP   |
|    |        |        |            | +0100 CET m=+0.001059140       |     |           |        |
+----+--------+--------+------------+--------------------------------+-----+-----------+--------+
stop bids
+----+------+-------+------------+------+-----+-----------+--------+
| ID | TYPE | PRICE | STOP PRICE | TIME | QTY | FILLEDQTY | PARAMS |
+----+------+-------+------------+------+-----+-----------+--------+
+----+------+-------+------------+------+-----+-----------+--------+
stop asks
+--------------------------------+-------+-------+-----+---------+-------+
|              TIME              | BIDID | ASKID | QTY |  PRICE  | TOTAL |
+--------------------------------+-------+-------+-----+---------+-------+
| 2021-02-20 01:15:03.838977142  |     3 |     2 |  40 | 24.0000 |   960 |
| +0100 CET m=+0.002366289       |       |       |     |         |       |
+--------------------------------+-------+-------+-----+---------+-------+
trades
Market price: 24.0000
enter instruction:instructions: [sell 20 limit 23.56]
+----+------+-------+------------+------+-----+-----------+--------+
| ID | TYPE | PRICE | STOP PRICE | TIME | QTY | FILLEDQTY | PARAMS |
+----+------+-------+------------+------+-----+-----------+--------+
+----+------+-------+------------+------+-----+-----------+--------+
bids
+----+-------+---------+------------+--------------------------------+-----+-----------+--------+
| ID | TYPE  |  PRICE  | STOP PRICE |              TIME              | QTY | FILLEDQTY | PARAMS |
+----+-------+---------+------------+--------------------------------+-----+-----------+--------+
|  4 | Limit | 23.5600 |     0.0000 | 2021-02-20 01:15:03.839842106  |  20 |         0 |        |
|    |       |         |            | +0100 CET m=+0.003231272       |     |           |        |
|  2 | Limit | 24.0000 |     0.0000 | 2021-02-20 01:15:03.838227556  |  50 |        40 |        |
|    |       |         |            | +0100 CET m=+0.001616705       |     |           |        |
+----+-------+---------+------------+--------------------------------+-----+-----------+--------+
asks
+----+--------+--------+------------+--------------------------------+-----+-----------+--------+
| ID |  TYPE  | PRICE  | STOP PRICE |              TIME              | QTY | FILLEDQTY | PARAMS |
+----+--------+--------+------------+--------------------------------+-----+-----------+--------+
|  1 | Market | 0.0000 |    25.0000 | 2021-02-20 01:15:03.837669981  |  20 |         0 | STOP   |
|    |        |        |            | +0100 CET m=+0.001059140       |     |           |        |
+----+--------+--------+------------+--------------------------------+-----+-----------+--------+
stop bids
+----+------+-------+------------+------+-----+-----------+--------+
| ID | TYPE | PRICE | STOP PRICE | TIME | QTY | FILLEDQTY | PARAMS |
+----+------+-------+------------+------+-----+-----------+--------+
+----+------+-------+------------+------+-----+-----------+--------+
stop asks
+--------------------------------+-------+-------+-----+---------+-------+
|              TIME              | BIDID | ASKID | QTY |  PRICE  | TOTAL |
+--------------------------------+-------+-------+-----+---------+-------+
| 2021-02-20 01:15:03.838977142  |     3 |     2 |  40 | 24.0000 |   960 |
| +0100 CET m=+0.002366289       |       |       |     |         |       |
+--------------------------------+-------+-------+-----+---------+-------+
trades
Market price: 24.0000
enter instruction:instructions: [buy 10 limit 26 FOK]
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
| 2021-02-20 01:15:03.838977142  |     3 |     2 |  40 | 24.0000 |   960 |
| +0100 CET m=+0.002366289       |       |       |     |         |       |
| 2021-02-20 01:15:03.840824027  |     5 |     4 |  10 | 26.0000 |   260 |
| +0100 CET m=+0.004213175       |       |       |     |         |       |
| 2021-02-20 01:15:03.840828196  |     1 |     4 |  10 | 23.5600 | 235.6 |
| +0100 CET m=+0.004217343       |       |       |     |         |       |
| 2021-02-20 01:15:03.84082885   |     1 |     2 |  10 | 24.0000 |   240 |
| +0100 CET m=+0.004217998       |       |       |     |         |       |
+--------------------------------+-------+-------+-----+---------+-------+
trades
Market price: 24.0000
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
