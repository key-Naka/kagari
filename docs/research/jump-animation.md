# 页面跳转动画源码调研

## 调研范围与结论

调研对象为 [JIEJOE-WEB-Tutorial/006-jump-animation](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation) 的 `main` 分支。仓库以同一视觉效果给出两套对照实现：

- `refresh/`：两个独立 HTML 文档，通过完整页面导航实现；
- `no refresh/`：Vue 2 单页应用（SPA），通过 Vue Router 实现无刷新导航。

两种方案的本质都是：用固定定位、全屏且高层级的 Loading 遮罩覆盖当前页；先执行遮罩进入动画，再提交跳转；目标页准备好后让遮罩向下退出。README 明确说明：普通 HTML 方案在浏览器历史前进/后退时无法触发进入动画，且目标页的进出动画不连续；Vue 无刷新方案可在浏览器历史跳转中触发，两个页面之间的动画连续。[来源：README](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/README.md)

## 关键文件

| 文件 | 作用 |
| --- | --- |
| [`README.md`](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/README.md) | 说明两种实现与其在刷新、历史跳转、动画连续性上的差异。 |
| [`refresh/page1.html`](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/refresh/page1.html)、[`page2.html`](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/refresh/page2.html) | 两个独立页面；各自内置相同的遮罩、样式与跳转脚本。 |
| [`no refresh/package.json`](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/no%20refresh/package.json) | Vue 2 / Vue Router 3 与 Vue CLI 的依赖和脚本定义。 |
| [`no refresh/src/main.js`](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/no%20refresh/src/main.js) | 创建 Vue 根实例，并注册路由。 |
| [`no refresh/src/App.vue`](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/no%20refresh/src/App.vue) | 始终挂载遮罩与路由视图；注册全局前置守卫。 |
| [`no refresh/src/components/loading.vue`](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/no%20refresh/src/components/loading.vue) | Loading 遮罩的 DOM、入场/退场控制和 CSS 动画。 |
| [`no refresh/src/router/index.js`](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/no%20refresh/src/router/index.js) | `/page1`、`/page2` 路由、根路径重定向及页面组件懒加载。 |
| [`no refresh/src/pages/page1.vue`](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/no%20refresh/src/pages/page1.vue)、[`page2.vue`](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/no%20refresh/src/pages/page2.vue) | 展示两种底色的全屏页面；文本点击后发起路由跳转。 |

## 核心实现

### 1. 遮罩是转场的唯一主体

遮罩占满视口，使用 `position: fixed` 与极高 `z-index`，因此能在旧页面和新页面之上持续存在。它的默认状态是显示在视口内；增加 `loading_out` 类后，以 `transform: translateY(100%)` 在 1 秒、`ease` 曲线中整体滑到视口下方，同时 SVG 和文字透明。移除该类会利用同一 CSS transition 将遮罩从下方滑回页面，形成进入效果。[来源：`loading.vue`](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/no%20refresh/src/components/loading.vue)

SVG 圆环独立运行 `circle_rotate`：3 秒、`ease-in`、无限循环；它同时旋转圆环并改变 `stroke-dashoffset`，产生描边转动的 Loading 视觉。这个循环和 1 秒的遮罩移动动画相互独立。

### 2. 刷新跳转：先覆盖，再修改 `location`

`refresh/page1.html` 与 `page2.html` 分别维护一个具有 `in(target)` 和 `out()` 方法的本地对象。点击页面中央文本时，内联点击处理器调用 `in()`：先移除退出类，使遮罩在 1 秒内覆盖画面；随后定时 1 秒，将 `window.location.href` 设为另一个 HTML 文件。目标文档的 `load` 事件调用 `out()`，让新文档中的另一份遮罩退出。

该实现能遮盖点击触发的完整页面跳转，但页面重载导致遮罩 DOM、动画状态和 SVG 动画都重新创建；`back` / `forward` 等浏览器历史操作绕过点击处理器，故无法在离开页前启动遮罩。这与 README 的限制说明一致。[来源：`page1.html`](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/refresh/page1.html)、[`page2.html`](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/refresh/page2.html)

### 3. 无刷新跳转：前置守卫延迟确认路由

Vue 版本将 `<Loading>` 放在根组件 `App.vue` 中，与 `<router-view>` 同级；路由切换时根组件不卸载，因此同一个遮罩节点可持续覆盖旧、新页面。应用挂载后注册 `$router.beforeEach`：每次导航先调用遮罩组件的 `in(next)`，该方法移除退出类并在 1 秒后调用守卫的 `next()`，才允许路由确认。随后它要求根组件检查加载状态并调用 `out()`。

根组件的检查逻辑每 300ms 轮询 `document.readyState`；文档为 `complete` 后调用遮罩退出。对客户端 SPA 路由而言，文档通常已经完成加载，所以路由确认后最多约 300ms 就开始 1 秒的遮罩退出。因此主时序是：

1. **点击或历史导航开始**：路由前置守卫被调用，遮罩从下方上移；
2. **约 1 秒**：守卫 `next()` 被调用，Vue Router 切换 `/page1` 与 `/page2` 的懒加载页面组件；
3. **随后 0–300ms**：`readyState` 轮询触发遮罩退出；
4. **约再 1 秒**：遮罩下移完成，新页面完全可见；Loading 圆环在遮罩可见期间持续旋转。

页面之间由遮罩持续遮挡，因而避免完整页面重载造成的视觉断层；全局前置守卫也会收到浏览器历史导航。Vue Router 文档说明，全局 `beforeEach` 在每次导航触发，导航在守卫完成前处于 pending；旧式第三个 `next` 参数仍受支持，但一次守卫流程中必须恰好调用一次。[来源：`App.vue`](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/no%20refresh/src/App.vue)、[`loading.vue`](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/no%20refresh/src/components/loading.vue)、[Vue Router 导航守卫文档](https://router.vuejs.org/guide/advanced/navigation-guards.html)

## 路由与触发方式

路由配置把 `/` 重定向到 `/page1`，并为 `/page1`、`/page2` 分别使用动态导入的页面组件。页面中心的文本分别调用 `$router.push('page2')` 与 `$router.push('page1')` 发起程序化导航；浏览器前进、后退则由 Vue Router 的全局守卫统一拦截并执行同样的遮罩进入流程。[来源：路由配置](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/no%20refresh/src/router/index.js)、[页面 1](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/no%20refresh/src/pages/page1.vue)、[页面 2](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/no%20refresh/src/pages/page2.vue)

## 依赖

`refresh/` 无构建步骤、无第三方库，只使用浏览器 DOM、事件与 CSS/SVG 动画。

`no refresh/` 的运行时依赖是 `vue` `^2.6.14`、`vue-router` `^3.5.2` 和 `core-js` `^3.8.3`；开发和构建使用 Vue CLI 5、Babel、ESLint 与 `vue-template-compiler`。没有使用 GSAP 或其他动画包，动画完全由 CSS transition/keyframes 完成。[来源：`package.json`](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/no%20refresh/package.json)

## 可迁移到 Nuxt 4 的建议

### 架构映射

1. 在 `app/app.vue` 中保留一个全局的、固定定位的 `PageLoadingOverlay`，并以 `<NuxtPage />` 渲染页面；不要在每个页面复制遮罩。`<NuxtPage>` 是 Nuxt 对 Vue Router 的封装，负责页面与内部状态处理，适合替代直接使用 `RouterView`。[来源：Nuxt `<NuxtPage>` 文档](https://nuxt.com/docs/4.x/api/components/nuxt-page)
2. 将原有两个页面分别迁移为 `app/pages/page1.vue`、`app/pages/page2.vue`，以 `<NuxtLink>` 处理声明式链接；需要根据业务代码决定目标时使用 `navigateTo()`。页面必须各自只有一个根元素，才可启用 Nuxt 的页面过渡。[来源：Nuxt 页面过渡文档](https://nuxt.com/docs/4.x/getting-started/transitions)
3. 遮罩的显示状态应由一个 composable 或 `useState` 统一管理；让页面、路由中间件和遮罩组件共享 `covering` / `leaving` 状态，而不是让子组件通过 `$parent` 调用方法或轮询 `document.readyState`。Nuxt 的页面组件在 `<Suspense>` 语义下，新页可能早于旧页卸载，因此全局覆盖层必须以状态机确保只有一次进入/退出动画。[来源：Nuxt `<NuxtPage>` 生命周期说明](https://nuxt.com/docs/4.x/api/components/nuxt-page)

### 导航流程与时序

- **推荐基础方案：** 对普通页面切换，直接采用 Nuxt `app.pageTransition`（例如 `mode: 'out-in'`）或为特定页配置 `definePageMeta({ pageTransition: ... })`。如果效果仍以 Loading 遮罩为中心，可使旧页离开动画与遮罩进入一致，并在新页准备完成后触发遮罩退出。Nuxt 官方支持全局、单页以及 JavaScript 钩子的页面过渡配置。
- **需要强制“先遮挡、再提交导航”时：** 将启动导航集中到客户端 composable：开始覆盖层 1 秒入场，等待 CSS `transitionend`（同时设置合理超时兜底），然后调用 `navigateTo()`；新页面完成必要的首屏数据准备后再开始遮罩 1 秒退场。不要用固定 `setTimeout` 作为唯一完成信号，以免用户设置减少动画、样式时长变更或页面性能波动时出现时序错位。
- **历史导航：** 浏览器后退/前进无法经过仅封装 `navigateTo()` 的点击处理，因此应另设全局路由中间件处理这些导航，或选择 Nuxt 原生页面过渡覆盖所有导航。中间件只能使用客户端可用的 DOM 动画控制逻辑；服务端渲染期间不应访问 `document` 或 `window`。
- **数据与错误：** 遮罩退出应以目标页面实际可展示为条件；若导航被中止或页面数据失败，应清理遮罩状态并呈现错误页/反馈，避免永久遮挡。Nuxt 官方也提示 View Transitions 在页面 setup 中执行数据获取时会冻结 DOM 更新，数据密集页面应审慎启用该实验性能力。[来源：Nuxt 页面过渡与 View Transitions 注意事项](https://nuxt.com/docs/4.x/getting-started/transitions)

### 动画与体验细节

- CSS 保留全屏遮罩的 `transform` 过渡和 SVG 描边动画即可，无须引入动画库；为 `prefers-reduced-motion: reduce` 缩短或关闭移动与圆环旋转，并让导航立即完成。
- 覆盖层应使用语义化的状态/可访问性属性：加载期间设置适当的 `aria-busy`，装饰性 SVG 不参与读屏；不要用可点击的 `<p>`，改用按钮或链接并保留键盘焦点样式。
- 原始实现每次路由后轮询文档加载状态；在 Nuxt 4 中应移除这一轮询，改用过渡钩子、页面 Suspense 完成时机或页面数据状态驱动，从而避免多次定时器和难以预测的 0–300ms 延迟。
- 如考虑原生 View Transitions API，Nuxt 4 提供实验性开关且默认尊重 `prefers-reduced-motion`；应先为不支持该 API 的浏览器保留 CSS/Vue Transition 回退，再评估与异步数据加载的兼容性。[来源：Nuxt View Transitions 文档](https://nuxt.com/docs/4.x/getting-started/transitions)

## 许可注意事项

仓库根目录采用 **MIT License**，版权信息为 `Copyright (c) 2025 JIEJOE'S WEB Tutorial`。MIT 允许使用、复制、修改、合并、发布、分发、再许可和销售，但在软件的副本或实质性部分中必须保留版权声明与许可文本，并按“按原样”免责条款提供。[来源：LICENSE](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/LICENSE)

本报告只归纳设计和机制，未复制完整源码。若迁移时复制了原仓库中具有实质性的 HTML、Vue、CSS 或 SVG 代码，应随副本保留上述 MIT 版权和许可声明；若只是按照本文重新实现“全屏遮罩—导航—退出”的通用模式，则仍应独立核实实际复用内容与项目自身的许可义务。

## 参考链接

- [上游仓库](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation)
- [README](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/README.md)
- [刷新方案：page1.html](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/refresh/page1.html)
- [刷新方案：page2.html](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/refresh/page2.html)
- [无刷新方案：Loading 组件](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/no%20refresh/src/components/loading.vue)
- [无刷新方案：根组件](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/no%20refresh/src/App.vue)
- [无刷新方案：路由配置](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/no%20refresh/src/router/index.js)
- [无刷新方案：依赖定义](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/no%20refresh/package.json)
- [MIT 许可证](https://github.com/JIEJOE-WEB-Tutorial/006-jump-animation/blob/main/LICENSE)
- [Nuxt 4 页面过渡文档](https://nuxt.com/docs/4.x/getting-started/transitions)
- [Nuxt 4 `<NuxtPage>` 文档](https://nuxt.com/docs/4.x/api/components/nuxt-page)
- [Vue Router 导航守卫文档](https://router.vuejs.org/guide/advanced/navigation-guards.html)
