# [1.0.0](https://github.com/omartelo/revu/compare/v0.2.1...v1.0.0) (2026-05-02)


* feat!: marca primeira release estável ([f9c3c8b](https://github.com/omartelo/revu/commit/f9c3c8bf0a11053d9ead5fd9f00bbbf61334cc40))


### Bug Fixes

* **frontend:** REV-44 bloquear merge quando checks IN_PROGRESS/PENDING/ERROR ([#13](https://github.com/omartelo/revu/issues/13)) ([541b8e1](https://github.com/omartelo/revu/commit/541b8e199f6118e7f769a8c6e9e84daaa41c9833))
* **hooks:** REV-45 pre-push checa drift no path correto frontend/src/generated ([#15](https://github.com/omartelo/revu/issues/15)) ([cd4500e](https://github.com/omartelo/revu/commit/cd4500e98b9237bc71ae8e64df5a28f43914e1d6))
* REV-43 dedup notificações com throttle + no-op em poll vazio ([#5](https://github.com/omartelo/revu/issues/5)) ([cd92574](https://github.com/omartelo/revu/commit/cd925747fca7758a1ae8cf31665e2aa0f394e48f))


### Features

* **frontend:** REV-33 adotar @tanstack/react-query como data layer ([#11](https://github.com/omartelo/revu/issues/11)) ([d2d13d5](https://github.com/omartelo/revu/commit/d2d13d5e6ad262caa08360a727694ac95a1fd2d5))
* REV-31 adotar @tanstack/react-router file-based no frontend ([#3](https://github.com/omartelo/revu/issues/3)) ([546f068](https://github.com/omartelo/revu/commit/546f068f5858052fc2cac742c9ea74b0ab911950))
* REV-38 ErrorBoundary global + por rota ([#7](https://github.com/omartelo/revu/issues/7)) ([c58bcb2](https://github.com/omartelo/revu/commit/c58bcb2a6697abe5f704f97ac47d171052aebf2f))
* REV-39 gerar constantes Go→TS pra eliminar drift ([#2](https://github.com/omartelo/revu/issues/2)) ([dbf8e8c](https://github.com/omartelo/revu/commit/dbf8e8cd31090b347d3f932b711a955a36d56112))
* REV-47 release automatizado via semantic-release ([#31](https://github.com/omartelo/revu/issues/31)) ([6e57ee2](https://github.com/omartelo/revu/commit/6e57ee25f21633b966ce97058bc56fd786806830))


### BREAKING CHANGES

* promove revu de fase MVP pra release estável (v1.0.0).
Sem mudanças técnicas — bump major manual via empty commit pra refletir
mudança de fase do projeto e disparar primeira run do pipeline de
release automatizado (REV-47).
