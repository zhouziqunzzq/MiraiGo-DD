# MiraiGo-DD
A DD (Japanese: 誰でも大好き) QQ Bot created based
on [MiraiGo](https://github.com/Mrs4s/MiraiGo) and
[MiraiGo-Template](https://github.com/Mrs4s/MiraiGo-Template).

[![Go Report Card](https://goreportcard.com/badge/github.com/zhouziqunzzq/MiraiGo-DD)](https://goreportcard.com/report/github.com/zhouziqunzzq/MiraiGo-DD)

## Modules
- logging: Copy from MiraiGo-Template. Provide basic logging for multiple events.
- auto_reconnect: Handle Disconnect event and try to reconnect.
- bili: A bilibili event broadcaster. It utilizes bilibili public API
  (ref [bilibili-API-collect](https://github.com/SocialSisterYi/bilibili-API-collect))
  to periodically poll subscribed user info and broadcast message if
  any event triggered by change of user status (e.g. start live streaming).
- daredemo_suki: Keyword-based random-memes sender.
- shell: Command-based interface for the bot. Configuring and querying bot
  status on the fly is under development.

## Configurations
Most of the config files are pretty much self-explained. You can always acquire
an example of them from xxx.example.yaml.

- application.yaml: Main config file for the app. Provide your account and password
  here as well as module-level configs for other modules.
- bili.yaml: Config file for bili module.
- dd.yaml: Config file for daredemo_suki module.
- shell.yaml: Config file for shell module.
- device.json: Config file for the simulated device info of the bot. If not provided,
  the app will randomly generate one at start. To avoid issue, it's recommended to
  use the same device.json among developing and production environments.

## Issues & PR
Feel free to share you thoughts in [Issues](https://github.com/zhouziqunzzq/MiraiGo-DD/issues),
and [PR](https://github.com/zhouziqunzzq/MiraiGo-DD/pulls) are highly welcomed.
