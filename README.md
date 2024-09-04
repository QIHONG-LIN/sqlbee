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

The SqlBee is based on your data struct, and this struct must follow the below two rules.

- The name of struct must be the same as the table name in the databse, with its first letter being capital. 
  For example, if you have a table `userInfo`, then your struct must named `UserInfo`.
- The struct must contain a field `Id int`, the position can be arbitrary.

    ```
    type UserInfo struct{
        //this is a must
        Id int
        //others
        Name string
        .....
    }
    ```

Wherever you want to do works with database, you first summon up a bee like this:
```
var user UserInfo
bee := sqlbee.SummonBeeFrom(user)
```


In SqlBee, almost all query/exec works are done by a **methods-chain**.

It starts with `bee.does`, and followed by so-called "middle methods", and the chain ends at the "end methods". The middle methods are used in th middle of the chain and they generally do not work with database, they're some help functions to form a final SQL query string. The end methods are those who really work with the database and carry out SQL to database.

```
bee.does.middle_methods().middle_methods()....end_methods()
```
We also provide direct method in a form of `bee.xxx()` instead of bee.does.xxx. This looks like we want the bee to do **a very simple job for struct** such as save() or delete(), see later sections.


### middle methods table

| middle method      | parameter | what is it? |
| ----------- | ----------- | ----------- |
| all()      |  none     | no filter condition, get all rows in the table. <br><br>Note: you can still apply other middle methods in the chain containing all(). However, it will make filter() loss its effect.|
| filter(_map)      |  _map `map[string]any`     | if map is {gender:'female'}, then it searches for all rows with gender = 'female' in the table.<br><br> The map can contain many elements, only make sure each key of the map exists in the database table.|
| exclude(_map)   |  _map `map[string]any`        |if map is {gender:'female'}, then it excludes all rows with gender = 'female' in the table. <br><br>The map can contain many elements, only make sure each key of the map exists in the database table.|
|order_by()|_order `string`|order_by("id") means the results will be in ascending order of 'id' to return; <br><br>order_by("-id") is in descending order.|

The end methods or direct methods please see later sections.

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

### end methods table for query

| middle method      | parameter | what is it? |
| ----------- | ----------- | ----------- |
| get()      | none     | do the sql query, take all rows you want and return them in `[]map[string]any`.|

## Insert/Update/Delete

These three frequent works are done in the same way. When you want to summon a bee you have to give a struct to it, so the bee knows what to care about. This struct can be empty, but it can also bee a filled struct and let you do something intuitive to human. 

1. If you want to **save/update** this data to database
    ```
    //a filled struct
    var user UserInfo
    user.Name = "qihong.lin"
    user.age = 18
    bee := sqlbee.SummonBeeFrom(user)

    // no Id>0, so user will be saved as a new row in the table.
    bee.save()

    // Id>0, so user will be updated into the corresponding row in the table.
    user.Id = 1
    bee := sqlbee.SummonBeeFrom(user)
    bee.save()
    ```
2.  If you want to delete this data from database
    ```
    //a filled struct
    //it can be from what you just queried.
    var user UserInfo
    user.Id = 1

    bee := sqlbee.SummonBeeFrom(user)
    bee.delete()
    ```
### direct methods table

| middle method      | parameter | what is it? |
| ----------- | ----------- | ----------- |
| save()      | none     | Save a row in database for the given struct if it has `'Id'==0`;<br><br>Or update the row corresponding to its 'Id' if `'Id'>0`.|
| delete()      | none     | Delete a row in database corresponding to the given struct's `Id` if `'Id'>0`. When `'Id'==0`, it will do nothing.|