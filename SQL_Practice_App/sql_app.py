import sqlite3

def create_connection(db_file):
    """ create a database connection to a SQLite database """
    conn = None
    try:
        conn = sqlite3.connect(db_file)
        print(f"Connected to SQLite database: {db_file}")
    except sqlite3.Error as e:
        print(e)
    return conn

def create_table(conn, create_table_sql):
    """ create a table from the create_table_sql statement
    :param conn: Connection object
    :param create_table_sql: a CREATE TABLE statement
    :return:
    """
    try:
        c = conn.cursor()
        c.execute(create_table_sql)
        print("Table created successfully.")
    except sqlite3.Error as e:
        print(f"Error creating table: {e}")

def add_table(conn, table_name, columns):
    columns_str = ", ".join([f"{col_name} {col_type}" for col_name, col_type in columns.items()])
    create_table_sql = f"CREATE TABLE IF NOT EXISTS {table_name} ({columns_str});"
    create_table(conn, create_table_sql)

def insert_data(conn, table_name, data):
    columns = ", ".join(data.keys())
    placeholders = ", ".join(["?" for _ in data.values()])
    sql = f"INSERT INTO {table_name} ({columns}) VALUES ({placeholders})"
    try:
        cur = conn.cursor()
        cur.execute(sql, tuple(data.values()))
        conn.commit()
        print("Data inserted successfully.")
    except sqlite3.Error as e:
        print(f"Error inserting data: {e}")

def select_all(conn, table_name):
    try:
        cur = conn.cursor()
        cur.execute(f"SELECT * FROM {table_name}")
        rows = cur.fetchall()
        return rows
    except sqlite3.Error as e:
        print(f"Error selecting data: {e}")
        return []

def update_data(conn, table_name, id, data):
    set_clause = ", ".join([f"{key} = ?" for key in data.keys()])
    sql = f"UPDATE {table_name} SET {set_clause} WHERE id = ?"
    try:
        cur = conn.cursor()
        cur.execute(sql, tuple(list(data.values()) + [id]))
        conn.commit()
        print("Data updated successfully.")
    except sqlite3.Error as e:
        print(f"Error updating data: {e}")

def delete_data(conn, table_name, id):
    sql = f"DELETE FROM {table_name} WHERE id = ?"
    try:
        cur = conn.cursor()
        cur.execute(sql, (id,))
        conn.commit()
        print("Data deleted successfully.")
    except sqlite3.Error as e:
        print(f"Error deleting data: {e}")

def main():
    database = "sql_practice.db"

    # create a database connection
    conn = create_connection(database)

    if conn:
        while True:
            print("\nSQL Practice App Menu:")
            print("1. Add a new table")
            print("2. Insert data")
            print("3. View all data")
            print("4. Update data")
            print("5. Delete data")
            print("6. Exit")
            choice = input("Enter your choice: ")

            if choice == '1':
                table_name = input("Enter table name: ")
                try:
                    num_columns = int(input("Enter number of columns: "))
                    columns = {}
                    for i in range(num_columns):
                        col_name = input(f"Enter name for column {i+1}: ")
                        col_type = input(f"Enter type for column {i+1} (e.g., TEXT, INTEGER, REAL): ")
                        columns[col_name] = col_type
                    add_table(conn, table_name, columns)
                except ValueError:
                    print("Invalid number of columns. Please enter an integer.")
            elif choice == '2':
                table_name = input("Enter table name to insert into: ")
                data = {}
                while True:
                    col_name = input("Enter column name (or 'done' to finish): ")
                    if col_name == 'done':
                        break
                    col_value = input(f"Enter value for {col_name}: ")
                    data[col_name] = col_value
                insert_data(conn, table_name, data)
            elif choice == '3':
                table_name = input("Enter table name to view: ")
                rows = select_all(conn, table_name)
                if rows:
                    for row in rows:
                        print(row)
                else:
                    print(f"No data found in {table_name} or table does not exist.")
            elif choice == '4':
                table_name = input("Enter table name to update: ")
                try:
                    id_to_update = int(input("Enter the ID of the row to update: "))
                    data = {}
                    while True:
                        col_name = input("Enter column name to update (or 'done' to finish): ")
                        if col_name == 'done':
                            break
                        col_value = input(f"Enter new value for {col_name}: ")
                        data[col_name] = col_value
                    update_data(conn, table_name, id_to_update, data)
                except ValueError:
                    print("Invalid ID. Please enter an integer.")
            elif choice == '5':
                table_name = input("Enter table name to delete from: ")
                try:
                    id_to_delete = int(input("Enter the ID of the row to delete: "))
                    delete_data(conn, table_name, id_to_delete)
                except ValueError:
                    print("Invalid ID. Please enter an integer.")
            elif choice == '6':
                break
            else:
                print("Invalid choice. Please try again.")

        conn.close()

if __name__ == '__main__':
    main()
