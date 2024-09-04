# 'sqlbee' for Golang
 ![sqlbee logo](https://img.picgo.net/2024/09/03/sqlbeed85db403a60f7923.png)

A simple sql package for Golang and gin. 


Author: @QIHONG-LIN

Contact email: linkinghone@163.com(Chinese) or qihong.lin22@imperial.ac.uk(International)

# How to use

1. Firstly, install the package into your project

    ```
    go get github.com/QIHONG-LIN/sqlbee
    ```

2. Secondly, create a json file `sqlbee_setting.json` and put it **in the root** of your project, and it should (only) contain the below content. You edit this json to fit your database need. 

    ```
    {
        "SqlBeeDb_username": "exampe_root",
        "SqlBeeDb_password": "exampe_123456",
        "SqlBeeDb_host": "exampe_10.10.10.10",
        "SqlBeeDb_port": "exampe_3306",
        "SqlBeeDb_dbname": "exampe_mySqlBeeDb"
    }
    ```

3. Finally, for example, in your main(), you can use this package like below.
    ```
    //create an instance based on your own data struct User
    bee := sqlbee.SummonBeeFrom(User{})
    //execute some commands and get results
	bag := bee.does.all().exclude(_map).order_by("-id").get()
    ```

# Tutorial

## Basic idea

In SqlBee, all query/exec are done by a methods-chain.

It starts with `bee.does`, and followed by so-called "middle methods", and the chain ends at the "end methods".

```
bee.does.middle_methods().middle_methods()....end_methods()
```

## Query

### 1. query with conditions

You start with `bee.does.` and choose your middle methods to form a query chain, and end with an end methods. In below example, `filter()`,  `exclude()`,  `order_by()` are middle methods, they only build up query string but not do the real query work. The end method `get()` execute the query string to get the results.

```
//example
bee.does.filter(_map_filter).exclude(_map_exclude).order_by("-id").get()
```
   
### 2. query all
```
bee.does.all().get()
```

### middle methods table

| middle method      | parameter | what is it? |
| ----------- | ----------- | ----------- |
| filter(_map)      |  _map `map[string]any`     | if map is {gender:'female'}, then it searches for all rows with gender = 'female' in the table|
| exclude(_map)   |  _map `map[string]any`        |if map is {gender:'female'}, then it excludes all rows with gender = 'female' in the table|