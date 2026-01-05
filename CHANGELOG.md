# Changelog

## 1.0.0 (2026-01-05)


### Features

* **auth:** add Gemini CLI authentication support ([9eecfba](https://github.com/GilmanLab/headjack/commit/9eecfba54e9da404b7a34ab749610d63f8252f8e)), closes [#26](https://github.com/GilmanLab/headjack/issues/26)
* **auth:** implement `hjk auth codex` command ([56e455e](https://github.com/GilmanLab/headjack/commit/56e455e1179e10c85d0a9a5bb15275531b80a071)), closes [#27](https://github.com/GilmanLab/headjack/issues/27)
* **auth:** implement Claude Code authentication and container session execution ([5af57ac](https://github.com/GilmanLab/headjack/commit/5af57ace64b399949ba9435472a4d27e5c3e5fbb)), closes [#20](https://github.com/GilmanLab/headjack/issues/20)
* **catalog:** add session tracking to schema ([89c4ffe](https://github.com/GilmanLab/headjack/commit/89c4ffe40086d4448c51735983899f8bef721244)), closes [#1](https://github.com/GilmanLab/headjack/issues/1)
* **ci:** add golangci-lint configuration and GitHub Actions workflow ([12649e7](https://github.com/GilmanLab/headjack/commit/12649e715af470c513ef715e29a49e0e091b5d36)), closes [#10](https://github.com/GilmanLab/headjack/issues/10)
* **ci:** add Homebrew tap automation via GoReleaser ([4739cd8](https://github.com/GilmanLab/headjack/commit/4739cd8cf454cf33e43aacd7bffc5dc8a072d958)), closes [#39](https://github.com/GilmanLab/headjack/issues/39)
* **ci:** add release-please for automated releases ([1f39da6](https://github.com/GilmanLab/headjack/commit/1f39da6cc725504a8d62aaaeff203a53c5bb1edf)), closes [#41](https://github.com/GilmanLab/headjack/issues/41)
* **cli:** add devcontainer integration support ([#49](https://github.com/GilmanLab/headjack/issues/49)) ([4d63859](https://github.com/GilmanLab/headjack/commit/4d6385910e05fb5152f00800504ae57b9824b9b4))
* **cli:** add passthrough flags for run and agent commands ([#56](https://github.com/GilmanLab/headjack/issues/56)) ([c8d4d13](https://github.com/GilmanLab/headjack/commit/c8d4d13b3c3a5e3ef8cbab53658b1abc9b447307))
* **cli:** add progress spinner during instance creation ([#60](https://github.com/GilmanLab/headjack/issues/60)) ([b3d64f4](https://github.com/GilmanLab/headjack/commit/b3d64f481dad22eac2cc6b1b6c78c436928b0ef9))
* **cmd:** add logs and kill commands for session management ([febbcc2](https://github.com/GilmanLab/headjack/commit/febbcc29d2dd68ef402239abfaac8a46220d4422)), closes [#17](https://github.com/GilmanLab/headjack/issues/17)
* **cmd:** rename list command to ps with session listing ([fecf20a](https://github.com/GilmanLab/headjack/commit/fecf20a3df64372b64fb711513a022b2cfe25d95)), closes [#16](https://github.com/GilmanLab/headjack/issues/16)
* **cmd:** rename new command to run with session support ([74d0b59](https://github.com/GilmanLab/headjack/commit/74d0b5947837010f50aefbd18c7607d16120b55b)), closes [#14](https://github.com/GilmanLab/headjack/issues/14)
* **cmd:** rename resume command to attach with MRU selection ([60037b8](https://github.com/GilmanLab/headjack/commit/60037b898d4a0b3df4d4a97acecca3d16bec8226)), closes [#15](https://github.com/GilmanLab/headjack/issues/15)
* **container:** add Docker as a container runtime option ([132dccc](https://github.com/GilmanLab/headjack/commit/132dccc9a07f16f415ac3cfed413fd72b5a62fc1)), closes [#36](https://github.com/GilmanLab/headjack/issues/36)
* **container:** add Docker runtime and integration tests ([bb80f7a](https://github.com/GilmanLab/headjack/commit/bb80f7a46c2d5484056f80850423266d561112a1)), closes [#37](https://github.com/GilmanLab/headjack/issues/37)
* **container:** add Podman runtime support with configurable settings ([a41f87c](https://github.com/GilmanLab/headjack/commit/a41f87c586bda9e825a74faec3fd731e8ddfe079)), closes [#22](https://github.com/GilmanLab/headjack/issues/22)
* **devcontainer:** add smart CLI fallback with local npm installation ([#55](https://github.com/GilmanLab/headjack/issues/55)) ([9261530](https://github.com/GilmanLab/headjack/commit/9261530dda2db9ba039495e4b81592591ffc084d))
* **features:** add devcontainer feature for AI coding agents ([#57](https://github.com/GilmanLab/headjack/issues/57)) ([e5b0c88](https://github.com/GilmanLab/headjack/commit/e5b0c88d0cd9b690a33a5b1576cf222b7921464e))
* **images:** add base image with CI/CD workflow ([ae4e5fe](https://github.com/GilmanLab/headjack/commit/ae4e5fee41810956cf2d740030e5657bf16cc04e)), closes [#5](https://github.com/GilmanLab/headjack/issues/5)
* **images:** add Zellij terminal multiplexer to base image ([183b02e](https://github.com/GilmanLab/headjack/commit/183b02e13a74ff5689c433ce4bf0f67ea5fa324f)), closes [#11](https://github.com/GilmanLab/headjack/issues/11)
* **instance:** add image label-based runtime configuration ([5bdce67](https://github.com/GilmanLab/headjack/commit/5bdce6760a6318fbed443ed3e7cce70d767bdf58)), closes [#23](https://github.com/GilmanLab/headjack/issues/23)
* **instance:** add session lifecycle management to Manager ([11826f7](https://github.com/GilmanLab/headjack/commit/11826f78f0c3f88e0cec31aaaf4d5176a81c1998)), closes [#12](https://github.com/GilmanLab/headjack/issues/12)
* **logging:** add session logging infrastructure ([b3ff2b0](https://github.com/GilmanLab/headjack/commit/b3ff2b09a73eed1434d8be826f27541fa25130bb)), closes [#4](https://github.com/GilmanLab/headjack/issues/4)
* **logging:** add structured logging with slog and charmbracelet/log ([#59](https://github.com/GilmanLab/headjack/issues/59)) ([13abf1d](https://github.com/GilmanLab/headjack/commit/13abf1df8742bb7a61509e3281cde430bff7b23a))
* **multiplexer:** add multiplexer integration layer ([8f61928](https://github.com/GilmanLab/headjack/commit/8f61928503bb70f3cc2ffc5b64dcf85a6ea5f896)), closes [#3](https://github.com/GilmanLab/headjack/issues/3)
* **multiplexer:** add tmux backend with configurable default ([e2e6b6d](https://github.com/GilmanLab/headjack/commit/e2e6b6d9d896e723999c91b5571cf3e8912483e8)), closes [#13](https://github.com/GilmanLab/headjack/issues/13)
* **names:** add Docker-style session name generator ([03ea15e](https://github.com/GilmanLab/headjack/commit/03ea15edadc4b9f69bea51550f3768aedd9a48f4)), closes [#2](https://github.com/GilmanLab/headjack/issues/2)


### Bug Fixes

* **ci:** consolidate image workflows to ensure sequential builds ([5fa1361](https://github.com/GilmanLab/headjack/commit/5fa1361c951195ee9c93afd68af117cf403c4339)), closes [#24](https://github.com/GilmanLab/headjack/issues/24)
* **ci:** correct release-please-action SHA ([44cd074](https://github.com/GilmanLab/headjack/commit/44cd074ad6934710bc970d473f2ff66a8554d049))
* **ci:** pin devcontainers/action to SHA ([a4f14b3](https://github.com/GilmanLab/headjack/commit/a4f14b3cdf04725663a93773e683c9c9ccfc5b13))
* **ci:** remove explicit output config from bake file ([d9498e1](https://github.com/GilmanLab/headjack/commit/d9498e1e394fce5382d73a2a8c15446db3439bfd))
* **ci:** use bake metadata output for image digests ([e80ac55](https://github.com/GilmanLab/headjack/commit/e80ac5506291848ae737feaa1fb6c54ef8b45362))
* **ci:** use bash lowercase for image name ([6d4346b](https://github.com/GilmanLab/headjack/commit/6d4346b3d0d201cabc4a106e7efb7e9d5a3315a1)), closes [#7](https://github.com/GilmanLab/headjack/issues/7)
* **ci:** use buildx bake for image builds with proper dependencies ([541d650](https://github.com/GilmanLab/headjack/commit/541d650e8b26e96cb088b982f037c0e653b25aa8)), closes [#25](https://github.com/GilmanLab/headjack/issues/25)
* **ci:** use correct format for multi-platform image digests ([493da0b](https://github.com/GilmanLab/headjack/commit/493da0bd4d99d2ef07a986ad9f560dc22c2a60dc))
* **cli:** improve run error messaging ([#46](https://github.com/GilmanLab/headjack/issues/46)) ([3f5da46](https://github.com/GilmanLab/headjack/commit/3f5da461b1c3881c1418aabdd3eb2e60f2008d9c))
* **container:** improve error handling and lifecycle management ([bfb7901](https://github.com/GilmanLab/headjack/commit/bfb7901bcae4bf5860c4d8f9aaa2e9ad63990f96)), closes [#18](https://github.com/GilmanLab/headjack/issues/18)
* **images:** add dind volumes ([#48](https://github.com/GilmanLab/headjack/issues/48)) ([84d3b39](https://github.com/GilmanLab/headjack/commit/84d3b39562e39ebb73b8404854de19e200e1e838))
* **images:** add docker cgroup flags ([#47](https://github.com/GilmanLab/headjack/issues/47)) ([69ef1b3](https://github.com/GilmanLab/headjack/commit/69ef1b372d208cf22feb7e889877e8663c771ad7))
* **images:** address security vulnerabilities in base image ([f2eb253](https://github.com/GilmanLab/headjack/commit/f2eb253ccd2e5b120e404dc17beee8abfddc93cf)), closes [#8](https://github.com/GilmanLab/headjack/issues/8)
* **images:** resolve Docker auto-start and systemd user issues ([a9f05f0](https://github.com/GilmanLab/headjack/commit/a9f05f0f7e041cfb4f005700f856621e3ed9a4a6)), closes [#9](https://github.com/GilmanLab/headjack/issues/9)
* **instance:** clean up exited sessions from catalog after attach ([1d10494](https://github.com/GilmanLab/headjack/commit/1d10494242b0db7f7fd6ca7fdcae98c90611f0ce)), closes [#21](https://github.com/GilmanLab/headjack/issues/21)
* **instance:** unify container shutdown logic to prevent orphaned resources ([522e7ed](https://github.com/GilmanLab/headjack/commit/522e7ed79b8330633fa734aba915fbc1ba7c188b)), closes [#19](https://github.com/GilmanLab/headjack/issues/19)


### Code Refactoring

* **auth:** implement cross-platform keyring and dual auth modes ([bdc96cb](https://github.com/GilmanLab/headjack/commit/bdc96cb965cd881b47ad9ca5be0c7d7ebe5fc8fd)), closes [#35](https://github.com/GilmanLab/headjack/issues/35)
* **cli:** separate instance creation from session management ([#54](https://github.com/GilmanLab/headjack/issues/54)) ([08d0d55](https://github.com/GilmanLab/headjack/commit/08d0d5568b68dbfed23d5bdad90eea53c454fff0))
* **cmd:** centralize manager/repo helpers and refactor CLI commands ([d02af5b](https://github.com/GilmanLab/headjack/commit/d02af5bdcd1374cc35a448046baa3023c1c5d50d)), closes [#30](https://github.com/GilmanLab/headjack/issues/30)
* **container:** consolidate runtime implementations into baseRuntime ([b12e688](https://github.com/GilmanLab/headjack/commit/b12e6883f1ca54c160af9882ea473337e3f0e039)), closes [#33](https://github.com/GilmanLab/headjack/issues/33)
* **container:** eliminate code duplication and improve quality ([ee8cda5](https://github.com/GilmanLab/headjack/commit/ee8cda5f681602f6c21bfe89c10e6994d3c546a6)), closes [#32](https://github.com/GilmanLab/headjack/issues/32)
* **container:** remove Apple Containerization support ([#50](https://github.com/GilmanLab/headjack/issues/50)) ([ebb321c](https://github.com/GilmanLab/headjack/commit/ebb321cdf486c4d6b8ee0478661d57df6ea423b8))
* **images:** remove systemd and dind image variants ([#52](https://github.com/GilmanLab/headjack/issues/52)) ([f194e0e](https://github.com/GilmanLab/headjack/commit/f194e0e879e5b4a193d3262f63be304aac269f22))
* **instance:** remove container image label support ([#51](https://github.com/GilmanLab/headjack/issues/51)) ([20c4e36](https://github.com/GilmanLab/headjack/commit/20c4e3656fe0c48cf80c0cb49a2529ec19223061))
* make devcontainers the default container mode ([#53](https://github.com/GilmanLab/headjack/issues/53)) ([8f8accf](https://github.com/GilmanLab/headjack/commit/8f8accf2b97426c77bf6619bcbf303fd56aa712b))
* **multiplexer:** remove Zellij support and standardize on tmux ([6782f26](https://github.com/GilmanLab/headjack/commit/6782f266e45b0e9ed282a486e0f372bc7b047bfb)), closes [#34](https://github.com/GilmanLab/headjack/issues/34)


### Miscellaneous

* add initial project files ([a98b89b](https://github.com/GilmanLab/headjack/commit/a98b89b0a71572b46cb60b6a8eff1e5a45a4c724))
* bootstrap project with CLI framework and container runtime ([25ad922](https://github.com/GilmanLab/headjack/commit/25ad922532221244d74a3c71f0659c0b35825c84))
* **cli:** refactor interface to introduce session-based design ([632e030](https://github.com/GilmanLab/headjack/commit/632e0303e6c8a0074d6cff3b56458ca8e948f32f))
* **cli:** refine defaults and agent auth handling ([15739bf](https://github.com/GilmanLab/headjack/commit/15739bfa4725b74c1264055acbc6c44568a430d2)), closes [#31](https://github.com/GilmanLab/headjack/issues/31)
* **config:** ignore CLAUDE.md file ([edbb696](https://github.com/GilmanLab/headjack/commit/edbb696885124b49f1876c42f610a3a04c7eae83))
* **config:** make Docker the default runtime and update docs for cross-platform support ([9135057](https://github.com/GilmanLab/headjack/commit/91350574ccf720090da0795ec15670e8a076a5f8)), closes [#38](https://github.com/GilmanLab/headjack/issues/38)

## Changelog

All notable changes to the Headjack CLI will be documented in this file.
