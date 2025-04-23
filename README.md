# Walkthrough
```
gqlsch -p <directory containing graphql, can be .ts .js>
```


## Manual Proses

### Query
1. choose 1 query to convert, say StockInventoryAdminList
2. look for the query stock_inventory(...) get the fields inside 
3. result no 2 find schema on raw hasura
    3.1. will be type stock_inventory { fetch only needed fields
4. put no 3 to wms-graph/graph/inventory.graphqls
    4.1. dependency bool_exp, e.g. field number
    4.2. dependency order_by, e.g. field created_by
    4.3. dependency distinct_on, e.g. field id
    4.4. dependency other object 
5. run make gqlgen
6. make build
7. push the to wms-graph repo origin
8. if there are some field not exist on the resolver implementation go back to step 4