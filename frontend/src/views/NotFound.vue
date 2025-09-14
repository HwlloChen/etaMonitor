<template>
	<div class="not-found">
		<mdui-icon name="error_outline" style="font-size: 80px; color: #f44336; margin-bottom: 24px;"></mdui-icon>
		<h1>404</h1>
		<p>页面未找到</p>
		<mdui-button variant="outlined" @click="$router.push('/')">返回首页</mdui-button>
	</div>
</template>

<style scoped>
.not-found {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	height: 80vh;
	color: rgb(var(--mdui-color-on-surface));
}

.not-found h1 {
	font-size: 48px;
	margin: 0 0 12px 0;
}

.not-found p {
	font-size: 18px;
	margin-bottom: 24px;
}
</style>
