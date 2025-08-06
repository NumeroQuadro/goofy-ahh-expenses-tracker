# Domain 
- project is telegram bot (with mini app feature) for tracking expenses that automatically counts saldo (particularly how much money i can spend each day based on expenses i wrote down)
- it should have a feature that allows me to upload my custom csv file (it has to has a particular structure, if not it has not be uploaded)
- mini-app has to has a simple ui: just calendar (with ability to switch month or year), a field for a transaction category (prepared multi-select options), field for a short description (optional) and field with amount (in rubles) and a button to submit a transaction
- the bot should send the file everyday in the evening (like 7pm)

# Technical details
- it should be written in golang
- all data should be stored in .csv file (no databases, no nosql solutions, no in-memory bases). i need this bot be quickly replacable (it should have a feature that allows me to get that .csv file with all the data, so i can edit it manually or move my system to another app with all my data saved)
- it has to have a convenient structure of a typical golang project: cmd, internal, config directories, Makefile (with commands docker-run, docker-build, docker-stop, docker-rm, docker-logs), Dockerfile, .env.example file (.env i will create for myself), .env file should be transfered to a Docker image during build stage.
- since a bot has a telegram mini-app support, it should support https (and certain .pem and other files should be moved to a docker contianer)
- you need to create unit tests for the solution, not super much, but main logic should be covered, especially parsers, validators. tests should be with t.Parallel and be in table style
- you do not need to create complicated tests (no e2e, no integrational tests)

