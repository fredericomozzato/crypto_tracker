import argparse
import os

from funcs_v2 import (
    valid_args,
    deposit,
    withdraw,
    update,
    erase,
    reset,
    print_table,
    refresh_values,
    create_new_portfolio,
)


if os.path.exists("./portfolio.csv"):
    pass
else:
    create_new_portfolio()


parser = argparse.ArgumentParser(
    prog="Crypto Tracker",
    description="CLI application to track your cryptocurrencies portfolio!",
)
parser.add_argument(
    "-d",
    "--deposit",
    nargs=2,
    required=False,
    help="Deposit a specified amount of the selected coin.",
    metavar=("ticker", "N"),
)
parser.add_argument(
    "-w",
    "--withdraw",
    nargs=2,
    required=False,
    help="Withdraw a specified amount of the selected coin.",
    metavar=("ticker", "N"),
)
parser.add_argument(
    "-u",
    "--update",
    nargs=2,
    required=False,
    help="Overwrites the value of the selected coin.",
    metavar=("ticker", "N"),
)
parser.add_argument(
    "-e",
    "--erase",
    nargs=1,
    required=False,
    help="Deletes the selected coin from the portfolio.",
    metavar="ticker",
)
parser.add_argument(
    "-r",
    "--reset",
    action="store_true",
    required=False,
    help="Starts a new portfolio from scratch. This action CANNOT BE UNDONE",
)

args = parser.parse_args()


def main() -> None:
    if args.deposit:
        valid_args(*args.deposit)
        deposit(args.deposit[0], args.deposit[1])

    elif args.withdraw:
        valid_args(*args.withdraw)
        withdraw(args.withdraw[0], args.withdraw[1])

    elif args.update:
        valid_args(*args.update)
        update(args.update[0], args.update[1])

    elif args.erase:
        erase(args.erase[0])

    elif args.reset:
        reset()

    else:
        refresh_values()
        print_table()


if __name__ == "__main__":
    main()
