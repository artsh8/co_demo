# скрипт для тестирования запросов с клиента
from collections.abc import Iterable
import asyncio
import random
import time
import httpx


async def unpack(response: httpx.Response) -> dict:
    await asyncio.sleep(0.1)
    try:
        res = response.json()
    except:
        res = {}
    return res


async def unpackl(respones: Iterable[httpx.Response]) -> list[dict]:
    results = []
    for r in respones:
        res = await unpack(r)
        results.append(res)
    return results


async def fetch_data(url: str) -> httpx.Response:
    async with httpx.AsyncClient() as client:
        return await client.get(url)


async def post_data(url: str, data: dict) -> httpx.Response:
    async with httpx.AsyncClient() as client:
        return await client.post(url, json=data)


async def dummy_amount() -> dict[str, int]:
    await asyncio.sleep(0.1)
    return {
        "customerOrders": random.randint(1, 10),
        "productOrders": random.randint(1, 15),
        "customers": random.randint(1, 3),
        "merchants": random.randint(1, 3),
        "products": random.randint(1, 5),
    }


async def add_data() -> httpx.Response:
    data = await dummy_amount()
    return await post_data("http://localhost:8080/gen/v1/dummy-data", data)


async def get_stats() -> httpx.Response:
    return await fetch_data("http://localhost:8080/gen/v1/stats")


async def chain():
    start = time.time()
    tasks = (
        add_data(),
        fetch_data("http://localhost:8080/api/v1/orders"),
        fetch_data("http://localhost:8080/api/v1/products/3"),
        fetch_data("http://localhost:8080/api/v1/products/5"),
        get_stats(),
    )
    responses = await asyncio.gather(*tasks)
    results = await unpackl(responses)

    tasks = (
        fetch_data("http://localhost:8080/api/v1/products"),
        fetch_data("http://localhost:8080/api/v1/restock"),
    )
    responses = await asyncio.gather(*tasks)
    results1 = await unpackl(responses)
    results.extend(results1)

    stats = await get_stats()
    stats = await unpack(stats)
    # stats = {}
    # stats["maxProduct"] = 100
    urls = [
        f"http://localhost:8080/api/v1/products/{id}"
        for id in range(1, stats["maxProduct"])
    ]
    tasks = [fetch_data(url) for url in urls]
    tasks1 = [fetch_data("http://localhost:8080/api/v1/orders") for _ in range(100)]
    tasks.extend(tasks1)
    responses = await asyncio.gather(*tasks)
    results2 = await unpackl(responses)
    results.extend(results2)
    end = time.time()
    # tail = results[:3]
    # for result in tail:
    #     print(result)

    print(f"Время выполнения: {end - start}")


if __name__ == "__main__":
    asyncio.run(chain())
