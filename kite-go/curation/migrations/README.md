# Migrations

For a description of the database migration workflow used at Kite: https://quip.com/XXXXXXX

1. Install migrate:

    ```
    go get github.com/mattes/migrate
    ```

2. Create a new migration

	```
    migrate -url $YOUR_LOCAL_DB -path . create name_of_your_change
    ```

3. Apply all available migrations to local (development) DB

	```
    migrate -url $YOUR_LOCAL_DB -path . up
    ```

More details: https://github.com/mattes/migrate
