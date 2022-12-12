import decimal
import csv
from typing import List, Tuple
import sys

from pycoingecko import CoinGeckoAPI  # type: ignore

from supported_coins import supported_coins


cg = CoinGeckoAPI()
PORTFOLIO_FILE: str = "portfolio.csv"


def valid_args(ticker: str, amount: str) -> bool:
    """
    Combines two functions to check if the user's input is correct
    :param ticker: ticker for the coin typed by the user
    :param amount: amount of the coin typed by the users
    :return: True if all arguments are correct of r False if one is wrong
    """
    if valid_coin(ticker) and valid_amount(amount):
        return True
    else:
        return False


def valid_coin(ticker: str) -> bool:
    """
    Checks if the coin typed by the user is supported by the program.

    The list of supported coins was parsed from the CoinGecko API
    and comprises the top 100 coins by market cap in november/2022.
    :param ticker: ticker of the coin typed by the user
    :return: True if the coin is supported. If not the program closes with
    an error message
    """
    for coin in supported_coins:
        if coin["symbol"] != ticker:
            continue
        else:
            return True
    sys.exit("ERROR: invalid coin")


def valid_amount(amount: str) -> bool:
    """
    Checks if the amount typed by the user is a number (int or float)
    and if it is positive
    :param amount: amount typed by the user
    :return: True if the amount is valid. If not the program closes with
    an error message
    """
    try:
        float(amount)
        if float(amount) < 0:
            sys.exit("ERROR: amount must be a positive number")
        else:
            return True
    except ValueError:
        sys.exit("ERROR: amount must be a positive number")


def read_csv(filename: str) -> List[dict]:
    """
    Reads the portfolio file to perform operations with them
    :param filename: name of the portfolio file
    :return: a list containing one dict for each coin in the portfolio
    """
    with open(filename, newline="") as readfile:
        reader = csv.DictReader(readfile)
        return [row for row in reader]


def write_csv(filename: str, values: List[dict]) -> None:
    """
    After any operation is done in with the values this function
    will write the modified data to the portfolio file
    :param filename: name of the portfolio file
    :param values: list of dicts with modifications
    :return: None
    """
    fieldnames: list = ["id", "ticker", "amount"]
    with open(filename, "w", newline="") as writefile:
        writer = csv.DictWriter(writefile, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(values)


def deposit(ticker: str, amount: str) -> None:
    portfolio: List[dict] = read_csv(PORTFOLIO_FILE)

    if in_portfolio(ticker):
        for coin in portfolio:
            if coin["ticker"] == ticker:
                coin["amount"] = decimal.Decimal(coin["amount"]) + decimal.Decimal(amount)
    else:
        portfolio.append({"ticker": ticker, "amount": amount, "id": get_coin_id(ticker)})

    write_csv(PORTFOLIO_FILE, portfolio)


def withdraw(ticker: str, amount: str) -> None:
    portfolio: List[dict] = read_csv(PORTFOLIO_FILE)

    if in_portfolio(ticker):
        for coin in portfolio:
            if coin["ticker"] == ticker:
                coin["amount"] = decimal.Decimal(coin["amount"]) - decimal.Decimal(amount)
                if coin["amount"] < 0:
                    sys.exit("ERROR: not enough funds to withdraw")

    write_csv(PORTFOLIO_FILE, portfolio)


def update(ticker: str, amount: str) -> None:
    portfolio: List[dict] = read_csv(PORTFOLIO_FILE)

    for coin in portfolio:
        if coin["ticker"] == ticker:
            coin["amount"] = amount

    write_csv(PORTFOLIO_FILE, portfolio)


def erase(ticker: str) -> None:
    portfolio: List[dict] = read_csv(PORTFOLIO_FILE)

    for coin in portfolio:
        if coin["ticker"] == ticker:
            portfolio.remove(coin)

    if input(f"ERASE {str(ticker).upper()}? (y/n)\n").lower() == "y":
        write_csv(PORTFOLIO_FILE, portfolio)
    else:
        sys.exit("Operation cancelled")


def reset() -> None:
    if input(f"RESET portfolio? (y/n)\n").lower() == "y":
        portfolio: List[dict] = read_csv(PORTFOLIO_FILE)

        for coin in portfolio:
            portfolio.remove(coin)

        write_csv(PORTFOLIO_FILE, portfolio)

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


def in_portfolio(ticker: str) -> bool:
    """
    Checks if a certain coin is already in the portfolio
    :param ticker: str with the ticker of the coin
    :return: True or False
    """
    portfolio = read_csv(PORTFOLIO_FILE)
    for row in portfolio:
        if row["ticker"] == ticker:
            return True
        else:
            continue
    return False


def get_coin_id(ticker: str) -> str:
    """
    Gets the id used to parse data from the CoinGecko API
    :param ticker: ticker of the coin
    :return: str with the coin's id
    """
    for coin in supported_coins:
        if coin["symbol"] == ticker:
            return coin["id"]
        else:
            continue


def get_values() -> Tuple[List[dict], Tuple[float, float]]:
    """
    Reads the data in the portfolio and calculate the values of each coin
    in USD and BRL. Adds these values to a dict and returns it
    :return: a list of dicts for each coin and a tuple with the total worth
    of the portfolio in USD and BRL
    """
    portfolio: List[dict] = read_csv(PORTFOLIO_FILE)
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
    """
    Requests the CoinGecko API for the current rate for each coin
    :param portfolio: list of dicts for each coin in portfolio
    :return: list of dicts with the current price for each coin in portfolio
    """
    return cg.get_price(ids=[coin["id"] for coin in portfolio], vs_currencies=["usd", "brl"])


def get_delta(portfolio: List[dict]) -> dict:
    """
    Requests the CoinGecko API fot the 24 hours variation in price
    for each coin in the portfolio
    :param portfolio: list of dicts for each coin in portfolio
    :return: dictionary with the coin and its 24 hours variation in price
    """
    market_data: dict = cg.get_coins_markets(vs_currency="usd", ids=[coin["id"] for coin in portfolio])
    return {coin["id"]: coin["price_change_percentage_24h"] for coin in market_data}


def get_totals(portfolio: List[dict]) -> Tuple[float, float]:
    """
    Sum the value for every coin in the portfolio to get the total
    value both in USD and BRL
    :param portfolio: list of dicts for each coin in portfolio
    :return: total value in USD and BRL
    """
    total_usd: float = 0.0
    total_brl: float = 0.0
    for coin in portfolio:
        total_usd += float(coin["usd_value"])
        total_brl += float(coin["brl_value"])
    return total_usd, total_brl


def create_new_portfolio() -> None:
    """
    It's only called if the program is being run for the first
    time in a system or if the portfolio file was deleted
    :return: None
    """
    fieldnames: List[str] = ["id", "ticker", "amount"]
    with open("portfolio.csv", "w") as writefile:
        writer = csv.DictWriter(writefile, fieldnames=fieldnames)
        writer.writeheader()


def main():
    pass


if __name__ == "__main__":
    main()
