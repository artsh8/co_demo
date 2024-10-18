import os
import psycopg2


db_params = {
    "dbname": os.getenv("PG_DBNAME"),
    "user": os.getenv("PG_USER"),
    "password": os.getenv("PG_PASSWORD"),
    "host": os.getenv("PG_HOST"),
    "port": os.getenv("PG_PORT"),
}


def init_schema():
    try:
        conn = psycopg2.connect(**db_params)
        cur = conn.cursor()
        cur.execute("SELECT 1 FROM customer")
        res = cur.fetchone()
        print("Схема уже существует")
        return None
    except psycopg2.errors.UndefinedTable as e:
        print(e)
    finally:
        cur.close()
        conn.close()
        print("Соединение для создания сехмы закрыто")

    try:
        conn = psycopg2.connect(**db_params)
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
