Final project for the CS50P course. This is a CLI cryptocurrency portfolio tracker built with Python by Frederico Mozzato in november, 2022.


The program runs entirely on the command line and stores its values on a csv file saved to the same directory where the python files are located. Its functionalities are simple, allowing you to deposit, withdraw and update values for up to 100 coins. It uses the CoinGecko API to retrieve realtime data from prices for all the supportde coins.


If you run it without any command line arguments it will display your current portfolio to the terminal.

$python crypto_tracker.py


To deposit coins to your portfolio you can run the --deposit (-d) command followed by the ticker of the coin you want to deposit and the quantity.

$python crypto_tracker.py --deposit btc 1


To withdraw funds from yout portfolio you can run the --withdraw (-w) command followed by the ticker you want to withdraw from and the quantity. If you try to withdraw more than the current value you have in your portfolio you will get an error.

$python crypto_tracker.py --withdraw btc 0.5


You can also update a coin value to overwrite  its value. Just run the --update (-u) command followed by the ticker of the coin you want to update and the new value you want it to be:

$python crypto_tracke.py --update btc 2.2


You can also delete a coin from yout porfolio by running the command --delete (-d) and passing the ticker of the coin to be deleted. The program will ask for a confirmation before it deletes the selected coin:

$python crypto_tracker.py --delete btc


You can also delete your whole portfolio and start from scratch with the --reset (-r) command. The program will ask for confirmation before reseting your portfolio:

$python crypto_tracker.py --reset