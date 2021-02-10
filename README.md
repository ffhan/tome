# Toy Order Matching Engine - TOME

TOME matches incoming buy/sell orders to create trades. It follows usual financial market order types, parameters and
market behaviour.

## CLI

As an example of the matching algorithm I've implemented a CLI that operates the order book.

`go run ./examples/cli/`

Example is provided at the end of the document.

## Currently supported

* order types
    * market order - execute an order as fast as possible, cross the spread
    * limit order - execute an order with a limit on bid/ask price (e.g. $x or less for a bid, or $y or more for an ask)
* order params
    * AON - all or nothing, don't allow partial fills
    * IOC - immediate or cancel, immediately fill what's possible, cancel the rest
    * FOK - AON+IOC, immediately match an order in full (without partial fills) or cancel it

## TODO

* stop orders
* GFD, GTC, GTD parameters
* logic surrounding the order book - trading hours, pre/after market restrictions
* basic middle & back office functionalities - risk assessment, limits
* TCP/UDP server that accepts orders
* reporting market volume, share price
* reporting acknowledgments & updates to clients (share price, displayed/hidden orders...)

## Market behaviour

Market orders are always given priority above all other orders, then sorted according to time of arrival.

* orders are FIFO based
    * bids - price (descending), time (ascending)
    * asks - price (ascending), time (ascending)
    * quantity does not matter in sorting

When a match occurs between two limit orders the price is set on the bid price. Bid of $25 and ask of $24 will be
matched at $25.

## Architecture (in development)

Order book & trade books are per-instrument objects, one order book can only handle one instrument.

* order book - stores active orders in memory, handles order matching
* trade book - stores daily trades in memory, provides additional data about trading
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

`BenchmarkOrderBook_Add-12    	  627396	      1877 ns/op	    1521 B/op	       2 allocs/op`

Each order insertion to the order book takes about 1.8 μs, which means we can (theoretically) match ~620k orders in ~1.2
seconds.

After all insertions 548066776 bytes (~548MB in use for about 69660 active orders - ~7.8kB/order) are in use, before
insertions 123231848 bytes. Reported allocations are really high per insertion, around 1.6kB - not acceptable & will
require memory profiling. About 78% of total memory usage comes through the Add method, 345% increase from the setup
state.

`cockroachdb/apd` gave a significant performance improvement over `shopspring/decimal` mainly because of huge memory
usage improvement which drastically lowered the allocation rates (from 44 alloc/op to 5 alloc/op).

## CLI example

1. buy 50 shares, market order, don't allow partial fills
1. sell 25 shares with a limit od 30, don't sell below that
    * previous buy order is not matched since it would be a partial fill
1. sell 55 shares with a limit of 25
    * matched with buy order from point 1 at our selling price of 25 - trade is entered
1. buy 30 shares with a limit of 25.5, don't buy above that + IOC
    * filled for 5 shares (order from point 3), the rest (25 shares) is cancelled and removed from the book

Example:

```
enter instruction:buy 50 market AON
instructions: [buy 50 market AON]
+----+--------+-------+--------------------------------+-----+-----------+--------+
| ID |  TYPE  | PRICE |              TIME              | QTY | FILLEDQTY | PARAMS |
+----+--------+-------+--------------------------------+-----+-----------+--------+
|  1 | Market |     0 | 2021-02-09 20:08:39.407451514  |  50 |         0 | AON    |
|    |        |       | +0100 CET m=+15.860611270      |     |           |        |
+----+--------+-------+--------------------------------+-----+-----------+--------+
+----+------+-------+------+-----+-----------+--------+
| ID | TYPE | PRICE | TIME | QTY | FILLEDQTY | PARAMS |
+----+------+-------+------+-----+-----------+--------+
+----+------+-------+------+-----+-----------+--------+
+------+-------+-------+-----+-------+-------+
| TIME | BIDID | ASKID | QTY | PRICE | TOTAL |
+------+-------+-------+-----+-------+-------+
+------+-------+-------+-----+-------+-------+
Market price: 20.25
enter instruction:sell 25 limit 30
instructions: [sell 25 limit 30]
+----+--------+-------+--------------------------------+-----+-----------+--------+
| ID |  TYPE  | PRICE |              TIME              | QTY | FILLEDQTY | PARAMS |
+----+--------+-------+--------------------------------+-----+-----------+--------+
|  1 | Market |     0 | 2021-02-09 20:08:39.407451514  |  50 |         0 | AON    |
|    |        |       | +0100 CET m=+15.860611270      |     |           |        |
+----+--------+-------+--------------------------------+-----+-----------+--------+
+----+-------+-------+--------------------------------+-----+-----------+--------+
| ID | TYPE  | PRICE |              TIME              | QTY | FILLEDQTY | PARAMS |
+----+-------+-------+--------------------------------+-----+-----------+--------+
|  2 | Limit |    30 | 2021-02-09 20:08:53.927406949  |  25 |         0 |        |
|    |       |       | +0100 CET m=+30.380566699      |     |           |        |
+----+-------+-------+--------------------------------+-----+-----------+--------+
+------+-------+-------+-----+-------+-------+
| TIME | BIDID | ASKID | QTY | PRICE | TOTAL |
+------+-------+-------+-----+-------+-------+
+------+-------+-------+-----+-------+-------+
Market price: 20.25
enter instruction:sell 55 limit 25
instructions: [sell 55 limit 25]
+----+------+-------+------+-----+-----------+--------+
| ID | TYPE | PRICE | TIME | QTY | FILLEDQTY | PARAMS |
+----+------+-------+------+-----+-----------+--------+
+----+------+-------+------+-----+-----------+--------+
+----+-------+-------+--------------------------------+-----+-----------+--------+
| ID | TYPE  | PRICE |              TIME              | QTY | FILLEDQTY | PARAMS |
+----+-------+-------+--------------------------------+-----+-----------+--------+
|  3 | Limit |    25 | 2021-02-09 20:09:22.686775031  |  55 |        50 |        |
|    |       |       | +0100 CET m=+59.139934743      |     |           |        |
|  2 | Limit |    30 | 2021-02-09 20:08:53.927406949  |  25 |         0 |        |
|    |       |       | +0100 CET m=+30.380566699      |     |           |        |
+----+-------+-------+--------------------------------+-----+-----------+--------+
+--------------------------------+-------+-------+-----+-------+-------+
|              TIME              | BIDID | ASKID | QTY | PRICE | TOTAL |
+--------------------------------+-------+-------+-----+-------+-------+
| 2021-02-09 20:09:22.686787487  |     1 |     3 |  50 |    25 |  1250 |
| +0100 CET m=+59.139947160      |       |       |     |       |       |
+--------------------------------+-------+-------+-----+-------+-------+
Market price: 25
enter instruction:buy 30 limit 25.5 IOC
instructions: [buy 30 limit 25.5 IOC]
+----+------+-------+------+-----+-----------+--------+
| ID | TYPE | PRICE | TIME | QTY | FILLEDQTY | PARAMS |
+----+------+-------+------+-----+-----------+--------+
+----+------+-------+------+-----+-----------+--------+
+----+-------+-------+--------------------------------+-----+-----------+--------+
| ID | TYPE  | PRICE |              TIME              | QTY | FILLEDQTY | PARAMS |
+----+-------+-------+--------------------------------+-----+-----------+--------+
|  2 | Limit |    30 | 2021-02-09 20:08:53.927406949  |  25 |         0 |        |
|    |       |       | +0100 CET m=+30.380566699      |     |           |        |
+----+-------+-------+--------------------------------+-----+-----------+--------+
+--------------------------------+-------+-------+-----+-------+-------+
|              TIME              | BIDID | ASKID | QTY | PRICE | TOTAL |
+--------------------------------+-------+-------+-----+-------+-------+
| 2021-02-09 20:09:22.686787487  |     1 |     3 |  50 |    25 |  1250 |
| +0100 CET m=+59.139947160      |       |       |     |       |       |
| 2021-02-09 20:09:57.343947775  |     4 |     3 |   5 |  25.5 | 127.5 |
| +0100 CET m=+93.797107453      |       |       |     |       |       |
+--------------------------------+-------+-------+-----+-------+-------+
Market price: 25.5
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
