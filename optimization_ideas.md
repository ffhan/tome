* [ ] reintroduce apd.Decimal in OrderTracker
    * why - apd.Decimal.Float64() currently consumes 18% of total CPU time (because it converts it to string and then parses the float)
    * why not - huge number of comparisons between apd.Decimal.Cmp might be a lot slower than a single Float64 call (followed by normal float64 comparisons)
