import type { Lang } from './site';

// Every word the site shows, in English and Chinese. Add both together;
// Chinese headings, titles and kickers omit the trailing 。 and body
// sentences keep it.

const en = {
  meta: {
    homeTitle: 'Scenra — Design apps by talking',
    homeDescription:
      'A desktop flow designer with an AI assistant. Describe an app and watch its screens and flow charts take shape on a canvas you can keep editing.',
    privacyTitle: 'Privacy',
    privacyDescription: 'What Scenra and its website count, and what they never send.',
  },
  brand: {
    name: 'Scenra',
    tagline: 'Screens and flow charts, drawn as you talk.',
  },
  nav: {
    features: 'Features',
    how: 'How it works',
    privacy: 'Privacy',
    github: 'GitHub',
    getStarted: 'Get started',
    menu: 'Menu',
    theme: 'Switch appearance',
    language: '简体中文',
    languageLabel: 'Read this page in Simplified Chinese',
  },
  actions: {
    github: 'View on GitHub',
    getStarted: 'Get started',
    skip: 'Skip to content',
    readme: 'Read the README',
    examples: 'Browse the examples',
    privacyNote: 'Read the privacy note',
    copy: 'Copy',
    copied: 'Copied',
    backHome: 'Back to home',
  },
  hero: {
    eyebrow: 'Open source · MIT · macOS, Windows and Linux',
    title: 'Design apps by talking.',
    lead: 'Scenra is a desktop designer with an AI assistant. Describe the app you have in mind, and watch its screens and flow charts take shape on a canvas you can keep editing.',
    primary: 'Get started',
    secondary: 'View on GitHub',
    note: 'Free and open source. Bring your own model.',
    imageAlt:
      'Scenra showing an order-handling flow chart on the canvas and the assistant in a side panel',
  },
  features: {
    kicker: 'Why Scenra',
    title: 'From first sentence to a file you can keep.',
    lead: 'Screens, flows and the conversation stay together, so a draft never gets lost between tools.',
    cards: [
      {
        icon: 'layers',
        title: 'Screens and flows in one place',
        body: 'Every .scenra file holds an app’s screens, each with a phone, tablet or desktop design and its flow charts, listed together in the sidebar.',
      },
      {
        icon: 'sparkles',
        title: 'An assistant that draws',
        body: 'The assistant asks what it needs, then lays out interfaces and adds, links and rearranges nodes through tools, not pasted text.',
      },
      {
        icon: 'palette',
        title: 'Designs without limits',
        body: 'Build screens from boxes, text, icons, images and SVG, shaped by Tailwind CSS classes. Zoom, pan and select elements like a canvas.',
      },
      {
        icon: 'pointer',
        title: 'Edits that follow your selection',
        body: 'Work on the element or node you select, read your other open files, or let the assistant read the project folder to chart how the code works.',
      },
      {
        icon: 'file',
        title: 'Plain text you own',
        body: 'No database, no lock-in. .scenra files are readable, diffable and editable by hand, and saves stay where you opened them.',
      },
      {
        icon: 'undo',
        title: 'Undo that makes sense',
        body: 'Each assistant change undoes as one step, so experiments are safe and nothing is left half-drawn.',
      },
    ],
  },
  canvases: {
    kicker: 'Two canvases, one conversation',
    title: 'Talk on the right. Draw in the middle.',
    lead: 'The assistant runs the full height of the window while designs and flow charts fill the canvas beside it. Select anything to point the conversation at it.',
    points: [
      'Phone, tablet and desktop designs share the canvas with every flow chart.',
      'Tabbed files keep their own conversation and undo history.',
      'The assistant reads your other open files, and the folder a file lives in.',
    ],
    preview: {
      design: 'Design',
      flow: 'Flow',
      screen: 'Sign in',
      welcome: 'Welcome back',
      email: 'you@example.com',
      button: 'Sign in',
      placed: 'Order placed',
      check: 'Check stock',
      reserve: 'Reserve every item',
      inStock: 'in stock',
      charge: 'Charge card',
      doing: 'In progress',
    },
  },
  model: {
    kicker: 'Bring your own model',
    title: 'Any OpenAI‑compatible provider.',
    lead: 'Use OpenAI, a local server, or anything that speaks the OpenAI API with tool calls. Your key and your conversation stay on your device, apart from the model you choose.',
    providers: ['OpenAI', 'Ollama', 'LM Studio', 'vLLM', 'Local servers'],
    note: 'Configure the base URL, key and model in Settings (⌘,).',
  },
  file: {
    kicker: 'Plain text',
    title: 'A file you can read, diff and commit.',
    lead: 'Every project is one .scenra file: plain text that sits beside the code it describes. Review it in a pull request, edit it by hand, or keep it in version control.',
  },
  privacy: {
    kicker: 'Statistics',
    title: 'Counted, not watched.',
    lead: 'Scenra counts how often it is used, nothing about what you make. This site does the same.',
    cards: [
      {
        icon: 'chart',
        title: 'In the app',
        body: 'Release builds send a self-hosted Umami a pageview at start, an event per message sent and heartbeats while you work, under a random install ID. No chart, message, file path or API key is sent. Turn it off in Settings → Statistics; dev builds and tests send nothing.',
      },
      {
        icon: 'globe',
        title: 'On this site',
        body: 'Pages are counted with Umami’s cookieless script, restricted to this site and the repository’s GitHub Pages address. No personal data, no advertising, no cross-site tracking.',
      },
    ],
    link: 'Read the privacy note',
  },
  start: {
    kicker: 'Get started',
    title: 'Designing in a few minutes.',
    lead: 'Scenra is free and open source. Install Go, clone the repository, and run it.',
    requirements: [
      'Go 1.26 or newer',
      'An OpenAI-compatible model with tool calls, hosted or local',
    ],
    terminal: 'Terminal',
    commands: [
      'git clone https://github.com/oneo-me/nsgo-scenra.git',
      'cd nsgo-scenra',
      'go tool nsgo dev',
    ],
    primary: 'Read the README',
    secondary: 'Browse the examples',
    note: 'Build macOS, Windows and Linux binaries with go tool nsgo build.',
  },
  faq: {
    title: 'Questions, answered.',
    items: [
      {
        q: 'Do I need an API key?',
        a: 'Only for hosted models. The app works with any OpenAI-compatible server, including one running on your machine.',
      },
      {
        q: 'Where are my designs kept?',
        a: 'In the .scenra files you choose. The app keeps its settings, the paths of open files and a random statistics ID on the device; nothing else.',
      },
      {
        q: 'Is there a web or mobile version?',
        a: 'No. Scenra is a desktop app for macOS, Windows and Linux.',
      },
      {
        q: 'What is sent for statistics?',
        a: 'Usage counts only: a pageview at start, an event per message and heartbeats while you work, under a random install ID with your platform. No chart, message, path or key. Turn it off in Settings → Statistics.',
      },
      {
        q: 'Can I use it for free?',
        a: 'Yes. Scenra is MIT licensed, and contributions are welcome under the same license.',
      },
    ],
  },
  footer: {
    product: 'Product',
    resources: 'Resources',
    elsewhere: 'Elsewhere',
    features: 'Features',
    how: 'How it works',
    privacy: 'Privacy',
    readme: 'README',
    changelog: 'Changelog',
    examples: 'Examples',
    blog: 'Blog',
    x: 'X',
    sponsors: 'GitHub Sponsors',
    rights: 'Scenra',
    built: 'Built with NSGo',
  },
  privacyPage: {
    title: 'Privacy',
    lead: 'Scenra is built to know as little about you as possible. This page covers both the app and this website.',
    sections: [
      {
        title: 'The app',
        body: [
          'Release builds send anonymous usage statistics to a self-hosted Umami instance, so that the project can tell how often and how long the app is used.',
        ],
        items: [
          'A pageview at start, one event for each message sent, and heartbeats while the chart or the conversation changes.',
          'A random install ID kept on the device, so installs are counted without identifying people. The platform, language and window size go with each event.',
          'No chart content, message, file path, folder name or API key is ever sent.',
          'Turn statistics off under Settings → Statistics. Builds made with go tool nsgo dev, and test binaries, send nothing.',
        ],
      },
      {
        title: 'This website',
        body: [
          'Pages are counted with the same self-hosted Umami instance, through a cookieless script tagged website.',
        ],
        items: [
          'It records page visits and clicks on a few outbound links, with the referring site, a coarse location derived from the request and the browser’s language.',
          'No cookies, fingerprints or advertising IDs are used, and nothing is shared with advertisers.',
          'The script runs only on the repository’s GitHub Pages address, never on local servers.',
        ],
      },
      {
        title: 'Local data',
        body: [],
        items: [
          'Settings, the paths of the files open in tabs and the random statistics ID live in the app’s own storage on your device.',
          'Scenra files are saved only where you choose.',
        ],
      },
      {
        title: 'Model providers',
        body: [
          'Prompts, the files you let the assistant read and the conversation are sent to the provider you configure, and are handled under that provider’s terms.',
        ],
        items: [],
      },
      {
        title: 'Contact',
        body: ['Questions or requests? Open an issue on GitHub.'],
        items: [],
      },
    ],
  },
  notFound: {
    title: 'Page not found',
    lead: 'The page you were looking for has moved or never existed.',
    home: 'Back to home',
  },
};

const zh: typeof en = {
  meta: {
    homeTitle: 'Scenra — 把应用设计，说出来',
    homeDescription:
      '带 AI 助手的桌面设计工具。说出你想要的应用，界面和流程图会在画布上成形，之后还能继续亲手修改。',
    privacyTitle: '隐私',
    privacyDescription: 'Scenra 及其网站统计什么，以及从不发送什么。',
  },
  brand: {
    name: 'Scenra',
    tagline: '把界面和流程，边说边画出来。',
  },
  nav: {
    features: '功能',
    how: '使用方式',
    privacy: '隐私',
    github: 'GitHub',
    getStarted: '开始使用',
    menu: '菜单',
    theme: '切换外观',
    language: 'English',
    languageLabel: 'Read this page in English',
  },
  actions: {
    github: '在 GitHub 查看',
    getStarted: '开始使用',
    skip: '跳到正文',
    readme: '阅读 README',
    examples: '浏览示例',
    privacyNote: '阅读隐私说明',
    copy: '复制',
    copied: '已复制',
    backHome: '返回首页',
  },
  hero: {
    eyebrow: '开源 · MIT 许可 · macOS、Windows 和 Linux',
    title: '把应用设计，说出来',
    lead: 'Scenra 是一款带 AI 助手的桌面设计工具。说出你脑海里的应用，界面和流程图就会在画布上成形，之后还能继续亲手修改。',
    primary: '开始使用',
    secondary: '在 GitHub 查看',
    note: '免费开源，带上自己的模型即可。',
    imageAlt: 'Scenra：画布上是一张订单处理流程图，右侧是助手面板',
  },
  features: {
    kicker: '为什么是 Scenra',
    title: '从第一句话，到一份留得住的文件',
    lead: '界面、流程和对话放在一起，草稿不会丢在几个工具之间。',
    cards: [
      {
        icon: 'layers',
        title: '界面与流程，在同一处',
        body: '每个 .scenra 文件保存应用的多个界面，每个界面都有手机、平板或桌面设计，以及一张或多张流程图，在侧栏中一起列出。',
      },
      {
        icon: 'sparkles',
        title: '会动手的助手',
        body: '助手会先问清需要什么，再通过工具排布界面、添加、连接和调整节点，而不是把一大段文字丢给你。',
      },
      {
        icon: 'palette',
        title: '不设限的设计',
        body: '用盒子、文字、图标、图片和 SVG 搭出界面，外观由 Tailwind CSS 类控制。像画布一样缩放、平移和选择元素。',
      },
      {
        icon: 'pointer',
        title: '改动跟着选择走',
        body: '只改你选中的元素或节点，也能读取其他打开的文件；需要时，还能读取文件所在的项目文件夹，把代码的运作方式画成流程。',
      },
      {
        icon: 'file',
        title: '纯文本，归你所有',
        body: '没有数据库，也没有厂商锁定。.scenra 文件可读、可 diff、可手改，保存时留在你打开它的位置。',
      },
      {
        icon: 'undo',
        title: '撤销，符合直觉',
        body: '助手的每次改动都作为一步撤销，放心试验，不会留下画到一半的东西。',
      },
    ],
  },
  canvases: {
    kicker: '一块对话，两块画布',
    title: '右边说，中间画',
    lead: '助手占据窗口整列高度，界面设计和流程图在旁边的画布上铺开。选中任意元素，对话就围绕它继续。',
    points: [
      '手机、平板和桌面设计，与所有流程图共享同一块画布。',
      '多个文件分标签打开，各自保留对话和撤销记录。',
      '助手能读取你其他打开的文件，以及文件所在的项目文件夹。',
    ],
    preview: {
      design: '设计',
      flow: '流程',
      screen: '登录',
      welcome: '欢迎回来',
      email: 'you@example.com',
      button: '登录',
      placed: '提交订单',
      check: '检查库存',
      reserve: '预留每件商品',
      inStock: '有库存',
      charge: '扣款',
      doing: '进行中',
    },
  },
  model: {
    kicker: '自带模型',
    title: '兼容任意 OpenAI 接口',
    lead: '使用 OpenAI、本地服务，或任何支持工具调用的 OpenAI 兼容接口。除了你选择的模型，密钥和对话都留在你的设备上。',
    providers: ['OpenAI', 'Ollama', 'LM Studio', 'vLLM', '本地服务'],
    note: '在设置（⌘,）中填写 Base URL、密钥和模型。',
  },
  file: {
    kicker: '纯文本',
    title: '一份能读、能 diff、能提交的文件',
    lead: '每个项目都是一个 .scenra 文件：纯文本，放在它所描述的代码旁边。可以在 pull request 里审阅、手写修改，或者纳入版本控制。',
  },
  privacy: {
    kicker: '使用统计',
    title: '被计数，不被监视',
    lead: 'Scenra 只统计使用情况，不统计你做了什么，网站也一样。',
    cards: [
      {
        icon: 'chart',
        title: '应用内',
        body: '正式构建会向自建的 Umami 发送匿名统计：启动时一次页面浏览，每发送一条消息一个事件，使用过程中发送心跳，全部使用设备上的随机安装 ID。不会发送流程图、消息、文件路径或 API 密钥。可在设置 → 统计中关闭；开发构建和测试不发送任何内容。',
      },
      {
        icon: 'globe',
        title: '网站上',
        body: '页面使用 Umami 的无 Cookie 脚本统计，只在本站和本仓库的 GitHub Pages 地址生效。没有个人数据，没有广告，也没有跨站跟踪。',
      },
    ],
    link: '阅读隐私说明',
  },
  start: {
    kicker: '开始使用',
    title: '几分钟内开始设计',
    lead: 'Scenra 免费开源。安装 Go，克隆仓库，就能运行。',
    requirements: [
      'Go 1.26 或更高版本',
      '任意支持工具调用的 OpenAI 兼容模型，云端或本地皆可',
    ],
    terminal: '终端',
    commands: [
      'git clone https://github.com/oneo-me/nsgo-scenra.git',
      'cd nsgo-scenra',
      'go tool nsgo dev',
    ],
    primary: '阅读 README',
    secondary: '浏览示例',
    note: '运行 go tool nsgo build 可生成 macOS、Windows 和 Linux 版本。',
  },
  faq: {
    title: '常见问题',
    items: [
      {
        q: '需要 API 密钥吗？',
        a: '只有云端模型需要。应用也兼容任何 OpenAI 接口的服务器，包括运行在本机上的。',
      },
      {
        q: '设计保存在哪里？',
        a: '保存在你选择的 .scenra 文件里。应用只在设备上保存设置、打开过的文件路径和一个随机的统计 ID。',
      },
      {
        q: '有网页版或移动版吗？',
        a: '没有。Scenra 是面向 macOS、Windows 和 Linux 的桌面应用。',
      },
      {
        q: '统计会上传什么？',
        a: '只有使用次数：启动时一次页面浏览、每条消息一个事件，以及使用过程中的心跳，附带随机安装 ID 和平台。不会上传流程图、消息、路径或密钥。可在设置 → 统计中关闭。',
      },
      {
        q: '可以免费使用吗？',
        a: '可以。Scenra 采用 MIT 许可，欢迎以相同许可贡献代码。',
      },
    ],
  },
  footer: {
    product: '产品',
    resources: '资源',
    elsewhere: '其他',
    features: '功能',
    how: '使用方式',
    privacy: '隐私',
    readme: 'README',
    changelog: '更新日志',
    examples: '示例',
    blog: '博客',
    x: 'X',
    sponsors: 'GitHub Sponsors',
    rights: 'Scenra',
    built: '基于 NSGo 构建',
  },
  privacyPage: {
    title: '隐私',
    lead: 'Scenra 尽量少知道关于你的事。本页同时说明应用和网站的做法。',
    sections: [
      {
        title: '应用',
        body: ['正式构建会向自建的 Umami 发送匿名使用统计，以便了解应用被使用的频率和时长。'],
        items: [
          '启动时一次页面浏览，每条消息一个事件，流程图或对话发生变化期间发送心跳。',
          '设备上保存一个随机安装 ID，用于统计安装数量而不识别个人。每个事件会带上平台、语言和窗口尺寸。',
          '不会发送任何流程图内容、消息、文件路径、文件夹名称或 API 密钥。',
          '可在设置 → 统计中关闭。使用 go tool nsgo dev 的构建以及测试程序都不发送任何内容。',
        ],
      },
      {
        title: '本网站',
        body: ['页面由同一个自建 Umami 统计，使用无 Cookie 脚本，并标记为 website。'],
        items: [
          '统计页面访问和对少量外链的点击，附带来源站点、由请求得出的粗略位置和浏览器语言。',
          '不使用 Cookie、浏览器指纹或广告 ID，也不与广告商共享任何内容。',
          '脚本只在本仓库的 GitHub Pages 地址运行，不会在本地服务器上运行。',
        ],
      },
      {
        title: '本地数据',
        body: [],
        items: [
          '设置、标签页中打开文件的路径和随机统计 ID 保存在应用自己的设备存储中。',
          'Scenra 文件只会保存到你选择的位置。',
        ],
      },
      {
        title: '模型服务商',
        body: ['提示词、你允许助手读取的文件以及对话会发送给你配置的服务商，并按该服务商的条款处理。'],
        items: [],
      },
      {
        title: '联系',
        body: ['有疑问或请求？请在 GitHub 上提 issue。'],
        items: [],
      },
    ],
  },
  notFound: {
    title: '页面不存在',
    lead: '你要找的页面已经移动，或从未存在。',
    home: '返回首页',
  },
};

export const copy = { en, zh } as const;

export type Copy = typeof en;
export type LangCopy = (typeof copy)[Lang];
