# Refreshing data

Data source used : 
- SO Data dump
    It is mostly used to have the view count. We need 2 dumps to be able to compute the delta of views as SO only give the total number of views and we need a monthly count. 
    The SO data dump have to be download manually and can then be fed to `so_dumps_processor` files. 
    
- Moz.com data:
    - That requires an account. The mostly used approach is to query Moz API to get all the queries leading to the 5k top python posts on SO. We need for the list of URL of the top 5k posts we want to scan. Then `moz_query.py` files gives tools to query Moz API for each URL and produce a csv file containing the list of queries leading to these pages. 
    - The second step is to process all these queries to produce a big python dict matching keywords with volume and SO page. That's done by `moz_processing.py` file. 
    
- GSC data
    We use GSC to get the list of queries bringing people to Kite.com. For that we use GSC API to get all the queries bringing people to any `/examples/` page. That's done with the `get_most_frequent_queries` function in `gsc_query` file. 
    For GSC you'll need to authenticate during the first execution of the program. Result of this authentication will be stored in the `web-content.dat` file. 
    
 

    
   