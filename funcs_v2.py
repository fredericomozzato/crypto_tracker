import decimal
import csv
from typing import List, Tuple
import sys

from pycoingecko import CoinGeckoAPI    # type: ignore

from supported_coins import supported_coins


cg = CoinGeckoAPI()
portfolio_file: str = "portfolio.csv"


def valid_args(ticker: str, amount: str) -> bool:
    if valid_coin(ticker) and valid_amount(amount):
        return True
    else:
        return False


def valid_coin(ticker: str) -> bool:
    for coin in supported_coins:
        if coin["symbol"] != ticker:
            continue
        else:
            return True
    sys.exit("ERROR: invalid coin")


def valid_amount(amount: str) -> bool:
    try:
        float(amount)
        if float(amount) < 0:
            sys.exit("ERROR: amount must be a positive number")
        else:
            return True
    except ValueError:
        sys.exit("ERROR: amount must be a positive number")


"""
The next two functions are to deal with all the context management
of the csv module. Since this basic logic is used many times in
different places it was important to abstract it.
The reader function reads the csv and returns the values in a dict.
The writer function takes the processed values and write it back to the csv.
"""


def read_csv(filename: str) -> List[dict]:
    with open(filename, newline='') as readfile:
        reader = csv.DictReader(readfile)
        return [row for row in reader]


def write_csv(filename: str, values: List[dict]) -> None:
    fieldnames: list = ["id", "ticker", "amount"]
    with open(filename, "w", newline='') as writefile:
        writer = csv.DictWriter(writefile, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(values)


"""
The next section defines the 6 operational functions
"""


def deposit(ticker: str, amount: str) -> None:
    portfolio: List[dict] = read_csv(portfolio_file)

    if in_portfolio(ticker):
        for coin in portfolio:
            if coin["ticker"] == ticker:
                coin["amount"] = decimal.Decimal(coin["amount"]) + decimal.Decimal(amount)
    else:
        portfolio.append(
            {"ticker": ticker, "amount": amount, "id": get_coin_id(ticker)}
        )

    write_csv(portfolio_file, portfolio)


def withdraw(ticker: str, amount: str) -> None:
    portfolio: List[dict] = read_csv(portfolio_file)

    if in_portfolio(ticker):
        for coin in portfolio:
            if coin["ticker"] == ticker:
                coin["amount"] = decimal.Decimal(coin["amount"]) - decimal.Decimal(amount)
                if coin["amount"] < 0:
                    sys.exit("ERROR: not enough funds to withdraw")

    write_csv(portfolio_file, portfolio)


def update(ticker: str, amount: str) -> None:
    portfolio: List[dict] = read_csv(portfolio_file)

    for coin in portfolio:
        if coin["ticker"] == ticker:
            coin["amount"] = amount

    write_csv(portfolio_file, portfolio)


def erase(ticker: str) -> None:
    portfolio: List[dict] = read_csv(portfolio_file)

    for coin in portfolio:
        if coin["ticker"] == ticker:
            portfolio.remove(coin)

    if input(f"ERASE {str(ticker).upper()}? (y/n)\n").lower() == "y":
        write_csv(portfolio_file, portfolio)
    else:
        sys.exit("Operation cancelled")


def reset() -> None:
    if input(f"RESET portfolio? (y/n)\n").lower() == "y":
        portfolio: List[dict] = read_csv(portfolio_file)

        for coin in portfolio:
            portfolio.remove(coin)

        write_csv(portfolio_file, portfolio)

    else:
        sys.exit("Operation cancelled")


def print_table() -> None:
    portfolio, totals = get_values()

    print()
    print(" CRYPTO TRACKER ".center(83, "*"))
    print()
    print(
        "TICKER".ljust(10),
        "AMOUNT".ljust(13),
        "PRICE".rjust(6),
        "Δ 24H".rjust(10),
        "USD".rjust(10),
        "BRL".rjust(14),
        "%".rjust(11),
    )
    print("-".center(83, "-"))

    for coin in sorted(portfolio, key=lambda c: c["usd_value"], reverse=True):
        print(
            f"{coin['ticker'].upper():<5}",
            f"{float(coin['amount']):>11,.3f}",
            f"{float(coin['rates']['usd']):>13,.2f}",
            f"{coin['delta_24']:>10.3f}",
            f"{float(coin['usd_value']):>13,.2f}",
            f"{float(coin['brl_value']):>14,.2f}",
            f"{coin['%']:>10.2f}%",
        )

    print("-".center(83, "-"))
    print()
    print("TOTAL:".rjust(45), f"{totals[0]:>10,.2f}", f"{totals[1]:>14,.2f}")
    print()
    print("*".center(83, "*"))
    print()


"""
The next functions are used inside the operational ones to add
functionality and abstract away complexity
"""


def in_portfolio(ticker: str) -> bool:
    portfolio = read_csv(portfolio_file)
    for row in portfolio:
        if row["ticker"] == ticker:
            return True
        else:
            continue
    return False


def get_coin_id(ticker: str) -> str:
    for coin in supported_coins:
        if coin["symbol"] == ticker:
            return coin["id"]
        else:
            continue


def get_values() -> Tuple[List[dict], Tuple[float, float]]:
    portfolio: List[dict] = read_csv(portfolio_file)
    rates: List[dict] = get_rates(portfolio)
    deltas: dict = get_delta(portfolio)

    for coin in portfolio:
        coin["rates"] = rates[coin["id"]]
        coin["usd_value"] = float(coin["amount"]) * float(coin["rates"]["usd"])
        coin["brl_value"] = float(coin["amount"]) * float(coin["rates"]["brl"])
        coin["delta_24"] = deltas[coin["id"]]

    totals: Tuple[float, float] = get_totals(portfolio)

    for coin in portfolio:
        coin["%"] = (float(coin["usd_value"]) / totals[0]) * 100

    return portfolio, totals


def get_rates(portfolio: List[dict]) -> List[dict]:
    return cg.get_price(
        ids=[coin["id"] for coin in portfolio], vs_currencies=["usd", "brl"]
    )


def get_delta(portfolio: List[dict]) -> dict:
    market_data = cg.get_coins_markets(
        vs_currency="usd", ids=[coin["id"] for coin in portfolio]
    )
    return {coin["id"]: coin["price_change_percentage_24h"] for coin in market_data}


def get_totals(portfolio: List[dict]) -> Tuple[float, float]:
    total_usd: float = 0.0
    total_brl: float = 0.0
    for coin in portfolio:
        total_usd += float(coin["usd_value"])
        total_brl += float(coin["brl_value"])
    return total_usd, total_brl


def refresh_values() -> None:
    portfolio: List[dict] = read_csv(portfolio_file)
    write_csv(portfolio_file, portfolio)


def create_new_portfolio() -> None:
    fieldnames: List[str] = ["id", "ticker", "amount"]
    with open("portfolio.csv", "w") as writefile:
        writer = csv.DictWriter(writefile, fieldnames=fieldnames)
        writer.writeheader()


def main():
    pass


if __name__ == "__main__":
    main()
