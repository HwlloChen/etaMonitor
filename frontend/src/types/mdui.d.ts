declare module 'mdui' {
  // MDUI Web Components 类型声明
  const mdui: any;
  export = mdui;
}

// 为Vue模板添加MDUI组件的全局类型声明
declare global {
  namespace JSX {
    interface IntrinsicElements {
      'mdui-navigation-drawer': any;
      'mdui-navigation-drawer-item': any;
      'mdui-top-app-bar': any;
      'mdui-top-app-bar-title': any;
      'mdui-button': any;
      'mdui-button-icon': any;
      'mdui-card': any;
      'mdui-icon': any;
      'mdui-chip': any;
      'mdui-list': any;
      'mdui-list-item': any;
      'mdui-avatar': any;
      'mdui-text-field': any;
      'mdui-select': any;
      'mdui-menu-item': any;
      'mdui-dialog': any;
      'mdui-table': any;
      'mdui-segmented-button-group': any;
      'mdui-segmented-button': any;
    }
  }
}

export {}