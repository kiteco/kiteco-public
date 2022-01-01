# Train dataset generator

Process github crawl to extract records (block of 10 successive tokens) that end up or not by a keyword. 


## Frequencies and balancing

With scanning 10M files for collection and 10k max samples, that's the frequencies we get for keywords (the int before is the category of the keyword).

The 2 last number correspond to the number of name expression or keyword expression.

```$xslt
{map[1:9715 2:10335 3:9068 4:9554 5:9454 6:10068 7:9148 8:10176 9:10257 10:9644 11:10280 12:9185 13:10007 14:10104 15:9537 16:5758 17:10349 18:9805 19:10352 20:9836 21:9579 22:9843 23:9111 24:10586 25:9846 26:10756 27:9862 28:9484 29:4049 30:5379] {0 0} 281127 289121}

```


