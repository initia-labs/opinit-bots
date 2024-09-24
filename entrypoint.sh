#!/bin/sh

# Initialize the bot if necessary
if [ ! -f ~/.opinit/$BOT_NAME.json ]; then
    opinitd init $BOT_NAME
fi

# Start the bot
opinitd start $BOT_NAME
