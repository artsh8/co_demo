import psycopg2

def init_schema():
    try:
        conn = psycopg2.connect(
        dbname="co", user="customuser", password="custompassword", host="pgpool", port="5432"
    )
        cur = conn.cursor()
        cur.execute("SELECT 1 FROM customer")
        res = cur.fetchone()
        print("Схема уже существует")
    except psycopg2.errors.UndefinedTable as e:
        print(e)
    finally:
        cur.close()
        conn.close()
        print("Соединение для создания сехмы закрыто")

    try:
        conn = psycopg2.connect(
            dbname="co", user="customuser", password="custompassword", host="pgpool", port="5432"
        )
        cur = conn.cursor()
        with open("./server/schema.sql", "r") as f:
            data = f.read()
        cur.execute(data)
        conn.commit()
        print("Схема успешно создана")
    except Exception as e:
        print(e)
    finally:
        cur.close()
        conn.close()
        print("Соединение для создания сехмы закрыто")

if __name__ == "__main__":
    init_schema()