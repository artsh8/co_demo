from contextlib import contextmanager
from flask import Flask, jsonify, request
import psycopg2
from psycopg2.extensions import connection, cursor
from faker import Faker


app = Flask(__name__)
fake = Faker()
stats = {"maxCustomer": 1, "maxMerchant": 1, "maxCo": 1, "maxProduct": 1}
stats_updates = 0
db_params = {
    "dbname": "co",
    "user": "customuser",
    "password": "custompassword",
    "host": "pgpool",
    "port": "5432",
}


@contextmanager
def get_cursor():
    conn: connection = None
    try:
        conn = psycopg2.connect(**db_params)
        cur: cursor = conn.cursor()
        yield cur
        conn.commit()
    except Exception as e:
        print(f"Ошибка: {e}")
        if conn:
            conn.rollback()
    finally:
        if cur:
            cur.close()
        if conn:
            conn.close()


def update_stats() -> None:
    global stats, stats_updates
    with get_cursor() as cur:
        cur.execute("SELECT MAX(id) FROM customer")
        stats["maxCustomer"] = cur.fetchone()[0]
        cur.execute("SELECT MAX(id) FROM merchant")
        stats["maxMerchant"] = cur.fetchone()[0]
        cur.execute("SELECT MAX(id) FROM customer_order")
        stats["maxCo"] = cur.fetchone()[0]
        cur.execute("SELECT MAX(id) FROM product")
        stats["maxProduct"] = cur.fetchone()[0]

    stats_updates += 1


@app.route("/v1/stats")
def get_stats():
    update_stats()
    return jsonify(stats)


@app.route("/ping")
def ping():
    return jsonify({"message": "pong"})


def gen_customer() -> tuple[str, str, str | None]:
    first_name = fake.word()
    last_name = fake.word()
    email = fake.email() if fake.random_int(0, 1) < 0.5 else None
    return first_name, last_name, email


def gen_merchant() -> str:
    return fake.word()


def gen_product() -> tuple[int, int, int, str]:
    merchant_id = fake.random_int(1, stats["maxMerchant"])
    price = fake.random_int(1, 99_999)
    amount = fake.random_int(1, 99_999)
    name = fake.word()
    return merchant_id, price, amount, name


def gen_co() -> int:
    return fake.random_int(1, stats["maxCustomer"])


def gen_po() -> tuple[int, int, int]:
    order_id = fake.random_int(1, stats["maxCo"])
    product_id = fake.random_int(1, stats["maxProduct"])
    amount = fake.random_int(1, 999)
    return order_id, product_id, amount


def add_cos(num: int) -> None:
    orders = ((gen_co(),) for _ in range(num))
    with get_cursor() as cur:
        cur.executemany("INSERT INTO customer_order (customer_id) VALUES (%s)", orders)


def add_pos(num: int) -> None:
    products = (gen_po() for _ in range(num))
    with get_cursor() as cur:
        cur.executemany(
            """
            INSERT INTO product_order (order_id, product_id, amount)
            SELECT val.order_id, val.product_id, val.amount
            FROM (
                VALUES (%s, %s, %s)
            ) val (order_id, product_id, amount)
            JOIN customer_order co ON val.order_id = co.id 
            JOIN product p on val.product_id = p.id
            ON CONFLICT (order_id, product_id) DO NOTHING""",
            products,
        )


def add_merhants(num: int) -> None:
    merchants = ((gen_merchant(),) for _ in range(num))
    with get_cursor() as cur:
        cur.executemany("INSERT INTO merchant (name) VALUES (%s)", merchants)


def add_customers(num: int) -> None:
    customers = (gen_customer() for _ in range(num))
    with get_cursor() as cur:
        cur.executemany(
            "INSERT INTO customer (first_name, last_name, email) VALUES (%s, %s, %s)",
            customers,
        )


def add_products(num: int) -> None:
    products = (gen_product() for _ in range(num))
    with get_cursor() as cur:
        cur.executemany(
            "INSERT INTO product (merchant_id, price, amount, name) VALUES (%s, %s, %s, %s)",
            products,
        )


def is_valid(json: dict) -> bool:
    required_keys = (
        "customerOrders",
        "productOrders",
        "customers",
        "merchants",
        "products",
    )
    for key in required_keys:
        if key in json:
            value = json[key]
            if not isinstance(value, int) or value <= 0:
                return False

    return True


def add_dummy(data: dict) -> tuple[list, list]:
    added = []
    errors = []
    for key, value in data.items():
        match key:
            case "customerOrders":
                add_cos(value)
                added.append(key)
            case "productOrders":
                add_pos(value)
                added.append(key)
            case "customers":
                add_customers(value)
                added.append(key)
            case "merchants":
                add_merhants(value)
                added.append(key)
            case "products":
                add_products(value)
                added.append(key)
            case _:
                errors.append(key)
    return added, errors


@app.route("/v1/dummy-data", methods=["POST"])
def dummy_data():
    if not request.is_json:
        return jsonify({"error": "Запрос должен быть в формате JSON"}), 400

    data = request.get_json()
    if not is_valid(data):
        return jsonify({"error": "Некорректный JSON"}), 400

    if stats_updates % 5 == 0:
        update_stats()

    added, errors = add_dummy(data)
    if not added:
        return jsonify({"error": "Тестовые данные не были добавлены"}), 500
    elif errors:
        return (
            jsonify(
                {
                    "message": f"Успешно добавлены: {added}. Атрибуты не поддерживаются: {errors}"
                }
            ),
            200,
        )

    return jsonify({"message": "Тестовые данные успешно добавлены"}), 200


if __name__ == "__main__":
    app.run(debug=True, port=8082)
