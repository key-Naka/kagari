# 无限滚动相册动画源码调研

## 调研范围与结论

调研对象为 [JIEJOE-WEB-Tutorial/008-infinite-scrolling](https://github.com/JIEJOE-WEB-Tutorial/008-infinite-scrolling) 的 `main` 分支。该项目是一个单页、二维可拖拽的图片墙：用户按住鼠标拖动时，所有卡片随指针在水平和垂直方向平移；卡片越过容器的一侧后立即从对侧回绕，因此视觉上可无限移动。它并非监听浏览器原生滚动条，也没有动态加载更多内容。

README 同时明确指出，该版本直接操作大量 DOM 节点，滑动可能卡顿，并给出了同主题的 Canvas 实现作为性能替代方案。[来源：README](https://github.com/JIEJOE-WEB-Tutorial/008-infinite-scrolling/blob/main/README.md)

## 仓库与关键文件

| 文件 | 作用 |
| --- | --- |
| [`infinite scrolling.html`](https://github.com/JIEJOE-WEB-Tutorial/008-infinite-scrolling/blob/main/infinite%20scrolling.html) | 唯一的页面入口，包含 HTML、CSS、交互状态与拖拽/回绕逻辑。 |
| [`gsap.js`](https://github.com/JIEJOE-WEB-Tutorial/008-infinite-scrolling/blob/main/gsap.js) | 随仓库提交的 GSAP 3.10.4 浏览器构建文件，用于将卡片位移写入 `transform` 并管理补间。 |
| [`photos/`](https://github.com/JIEJOE-WEB-Tutorial/008-infinite-scrolling/tree/main/photos) | 28 张 PNG 示例资源；页面以 4 行 × 7 张静态标记组织它们。 |
| [`README.md`](https://github.com/JIEJOE-WEB-Tutorial/008-infinite-scrolling/blob/main/README.md) | 效果说明、视频入口及 DOM 性能限制说明。 |
| [`LICENSE`](https://github.com/JIEJOE-WEB-Tutorial/008-infinite-scrolling/blob/main/LICENSE) | 仓库项目源码的 MIT 许可证。 |

## 核心实现方法

### 1. 固定尺寸的二维图片墙

页面以一个绝对定位的 `.photos` 容器承载四个纵向堆叠的行；每行是横向 flex 布局，固定 7 张卡片。卡片尺寸、行高、行间距和卡片间距均采用以 `em` 表达的固定设计单位，局部将 `font-size` 设为 1px；容器整体再按视口宽度相对 1440px 的比例缩放。窄屏媒体查询仅提高这些设计单位的基准字号，避免卡片过小。

因此，布局不是响应式重排的网格，而是一个固定几何关系的画布式图片墙。容器在文档流中绝对定位，页面高度锁定为视口高度并隐藏溢出，视觉窗口中只显示处于可见范围的卡片。

### 2. 每张卡片维护“初始位置 + 累计位移”

初始化和每次窗口尺寸变化时，脚本读取所有卡片的 `offsetLeft`、`offsetTop`、卡片宽高和容器宽高；随后为每张卡片保存其 DOM 节点、初始坐标及独立的累计 `x/y` 位移。这使回绕判断可以相对原始格点进行，而无需变更 HTML 顺序或复制图片节点。

窗口缩放时，脚本会：

1. 重算容器、卡片尺寸与整体缩放比例；
2. 将卡片变换复位；
3. 重建每张卡片的初始坐标与位移状态。

这种做法实现简单，但缩放时会丢弃当前拖拽位置，图片墙回到初始状态。

### 3. 鼠标拖拽与缩放坐标换算

容器通过 `mousedown` 记录拖拽启用状态和上次鼠标坐标；`mousemove` 中若正在拖拽，则以本次坐标减去上次坐标得到增量；`mouseup` 与 `mouseleave` 终止拖拽。由于图片墙整体经过 CSS 缩放，鼠标位移会除以缩放比例后再累加，避免视觉缩放造成拖拽距离失真。

每次移动会遍历所有卡片，把相同的二维增量累加到各卡片自身位移，然后以 GSAP 更新对应元素的 CSS `transform: translate(...)`。代码在创建新动画前会终止该卡片尚未完成的动画，避免连续 `mousemove` 生成的补间相互竞争。正常移动使用约 1 秒的缓动，发生回绕时使用零时长，保证位置重置不出现跨屏动画。

### 4. 循环（回绕）机制

循环不依赖克隆行/列，也不维护无限大的坐标。其边界条件按单张卡片的原始位置判断：

- 当卡片左侧加累计水平位移大于容器宽度时，累计位移减去一个容器宽度；
- 当其左侧加位移小于负卡片宽度时，累计位移加上一个容器宽度；
- 纵向分别在顶部超过容器高度、底部小于负卡片高度时，减去或加上一个容器高度。

这相当于对卡片位置分别做以容器宽、高为周期的模运算。由于原始布局的每一行覆盖一个容器宽度、所有行连同间距覆盖一个容器高度，边缘移出的卡片会在对侧回到相同的格点序列，从而保持视觉连续。

**边界条件：** 当前实现每次仅修正一个容器周期，隐含前提是单次鼠标事件的位移小于容器尺寸；极端跳跃式指针移动可能仍留在边界外。其输入只覆盖鼠标事件，触屏和触控笔没有适配。

## 图片展示与交互细节

- 卡片容器为圆角、裁切溢出的固定尺寸区域；图片高度填满卡片，未显式设置宽度，因此依据原始宽高比显示，并由裁切框截取溢出部分。
- 图片禁用 `pointer-events` 和文本选择，确保拖拽事件由图片墙容器统一接收，且不会触发浏览器原生拖图。
- 卡片悬停时图片进行短时放大，形成局部浏览反馈；放大受容器裁切，不改变网格几何。
- 容器使用指针形状提示可操作，但原始实现使用的是普通 `pointer`，并未体现按住拖拽的 `grab/grabbing` 状态。
- 未提供键盘操作、焦点管理、图片替代文本、点击打开详情或减少动态效果偏好处理；将其用于产品页面时需补足可访问性。

## 依赖与许可

### 依赖

页面本身没有包管理文件或构建步骤，只有浏览器原生 HTML/CSS/DOM API 与本地引入的 GSAP。内置文件头标识为 **GSAP 3.10.4**，并说明其适用 GreenSock Standard License；因此即使上层仓库使用 MIT，仍应单独遵守 GSAP 的许可条款。[来源：GSAP 文件头](https://github.com/JIEJOE-WEB-Tutorial/008-infinite-scrolling/blob/main/gsap.js)

### 许可注意事项

仓库根目录声明 **MIT License，Copyright (c) 2025 JIEJOE'S WEB Tutorial**。MIT 允许使用、修改、分发和再许可，但在副本或实质性部分中必须保留版权与许可声明，并按“按原样”免责条款使用。[来源：LICENSE](https://github.com/JIEJOE-WEB-Tutorial/008-infinite-scrolling/blob/main/LICENSE)

本调研仅归纳设计与机制，不复制完整页面或完整动画代码。若迁移时复用其具有实质性的源码、图片或其副本，应保留 MIT 版权/许可文本；GSAP 文件或其源码不能仅因该仓库为 MIT 就被视为自动改为 MIT，建议改用从官方 npm 包安装的对应版本并核对其许可证。示例图片的单独权利归属未在 README 或 LICENSE 中另行说明，生产项目应替换为自有或已获授权的资源。

## 迁移到 Nuxt 4 + Tailwind CSS 4 的建议

### 组件边界与客户端运行

建议将图片墙封装为 `components/gallery/InfiniteGallery.client.vue`，页面只负责提供图片数据与业务入口。Nuxt 4 会自动导入 `components/` 中的组件；`.client.vue` 后缀使依赖 DOM 尺寸、`window`、指针事件和动画库的代码只在挂载后的浏览器执行，避免 SSR 期间访问浏览器对象。[来源：Nuxt 4 组件目录与客户端组件文档](https://nuxt.com/docs/4.x/directory-structure/app/components)

组件内应使用 Vue 的模板渲染图片列表，而非手写 28 个节点：图片数组包含稳定 `id`、`src`、`alt`，按行列索引映射为初始位置；实际元素用模板 ref 收集。初始化测量、`ResizeObserver`、事件监听和 GSAP 实例须在 `onMounted` 中建立，并在 `onBeforeUnmount` 中清理监听、观察器及动画，以避免路由切换后的内存泄漏。

### 推荐的交互调整

1. **统一 Pointer Events：** 以 `pointerdown`、`pointermove`、`pointerup`、`pointercancel` 取代仅鼠标事件；在按下时调用 `setPointerCapture`，在元素外松开也能可靠结束拖动。容器添加 `touch-action: none`，同时支持鼠标、触控和触控笔。
2. **使用模块化位移：** 保存单个全局 `offsetX/offsetY` 或每卡片派生位移，以周期取模把坐标规范到合理区间；用真正的模运算处理单次跨越多个周期的情况。渲染位置通过“初始坐标 + 规范化位移”计算，避免只加减一次的边界缺口。
3. **绘制节流：** 指针事件只记录最新增量，在 `requestAnimationFrame` 中每帧统一写一次 transform；不要为每个 `pointermove` 和每个卡片创建一段一秒补间。若保留 GSAP，优先用 `gsap.quickSetter` 或单个帧循环写 transform；拖拽结束后才按需要添加惯性或缓动。
4. **缩放保持位置：** `ResizeObserver` 重新计算单元尺寸后，不应清空累计偏移；只更新周期和基准位置。可使用 CSS `clamp()`、`vw` 或容器查询代替原项目基于 1440px 的整体 `scale()`，以减少坐标换算。
5. **可访问性与体验：** 将仅装饰性图片标记为空 `alt`，内容图片提供语义 `alt`；可交互卡片使用按钮/链接并提供键盘焦点样式。使用 `prefers-reduced-motion` 降低悬停缩放与惯性动画。拖拽光标使用 `cursor-grab` / `cursor-grabbing`。

### Tailwind CSS 4 落地方式

Tailwind 官方的 Nuxt 安装指引要求安装 `tailwindcss` 与 `@tailwindcss/vite`，在 `nuxt.config.ts` 注册 Vite 插件，创建全局 CSS 入口并以 `@import "tailwindcss";` 引入，再在 Nuxt `css` 配置中声明该文件。[来源：Tailwind CSS：Nuxt 安装](https://tailwindcss.com/docs/installation/framework-guides/nuxt)

静态样式可迁移为工具类：全屏黑色舞台、溢出裁切、flex 行列、圆角、图片裁切、悬停缩放、选择禁止和响应式尺寸均适合直接表达。由测量结果计算的平移、单元几何和周期仍应通过 Vue 绑定的 `style.transform`、CSS 自定义属性或动画库写入；不要拼接不可预测的 Tailwind 类名，以确保构建时能识别类名。

### 性能选型

小规模图片墙（如本仓库的 28 张）可使用 DOM + transform，并通过每帧批量更新达到可接受效果。图片数量明显增加、需要持续惯性动画或低端移动设备流畅度时，应采用 Canvas/WebGL 或做可视区域虚拟化；这也与原 README 指向 Canvas 版本作为 DOM 卡顿替代方案的建议一致。[来源：README 的 Canvas 参考](https://github.com/JIEJOE-WEB-Tutorial/008-infinite-scrolling/blob/main/README.md)

## 参考链接

- [上游仓库](https://github.com/JIEJOE-WEB-Tutorial/008-infinite-scrolling)
- [页面源码](https://github.com/JIEJOE-WEB-Tutorial/008-infinite-scrolling/blob/main/infinite%20scrolling.html)
- [README](https://github.com/JIEJOE-WEB-Tutorial/008-infinite-scrolling/blob/main/README.md)
- [MIT 许可证](https://github.com/JIEJOE-WEB-Tutorial/008-infinite-scrolling/blob/main/LICENSE)
- [GSAP 文件及许可头](https://github.com/JIEJOE-WEB-Tutorial/008-infinite-scrolling/blob/main/gsap.js)
- [Nuxt 4 组件文档](https://nuxt.com/docs/4.x/directory-structure/app/components)
- [Tailwind CSS 的 Nuxt 安装文档](https://tailwindcss.com/docs/installation/framework-guides/nuxt)
