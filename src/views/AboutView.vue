<template>
  <div class="min-h-screen bg-gradient-to-br from-blue-50/50 via-white to-purple-50/30">
    <div class="max-w-4xl mx-auto px-3 sm:px-5 md:px-8 py-6 sm:py-10">
      <div class="bg-white/80 backdrop-blur-sm rounded-2xl shadow-lg shadow-black/5 border border-white/50 overflow-hidden">
        <div class="flex items-center px-3 sm:px-4 py-2 sm:py-2.5 bg-gradient-to-b from-black/8 to-black/2 md:hidden">
          <div class="w-3 h-3 rounded-full bg-[#ff5f57] mr-2"></div>
          <div class="w-3 h-3 rounded-full bg-[#febc2e] mr-2"></div>
          <div class="w-3 h-3 rounded-full bg-[#28c840] mr-4"></div>
          <div class="text-xs sm:text-sm text-gray-500 font-medium">关于我们</div>
        </div>

        <div class="p-4 sm:p-6 md:p-8">
          <div class="text-center mb-6 sm:mb-8">
            <div class="w-14 h-14 sm:w-16 sm:h-16 md:w-20 md:h-20 mx-auto mb-4 sm:mb-6 bg-gradient-to-br from-blue-500 to-purple-600 rounded-xl flex items-center justify-center shadow-lg shadow-blue-500/30">
              <svg class="w-7 h-7 sm:w-8 sm:h-8 md:w-10 md:h-10 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10" />
                <path d="M12 6v6l4 2" />
              </svg>
            </div>
            <h1 class="text-xl sm:text-2xl md:text-3xl font-bold text-gray-900 mb-2 sm:mb-3">小泉动漫</h1>
            <p class="text-sm sm:text-base text-gray-600">发现精彩二创内容，分享创作乐趣</p>
          </div>

          <div class="space-y-4 sm:space-y-6">
            <section>
              <h2 class="text-base sm:text-lg font-semibold text-gray-800 mb-3 flex items-center gap-2">
                <svg class="w-4 h-4 sm:w-5 sm:h-5 text-blue-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M12 20h9" />
                  <path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7.5 18H4v-3.5a2.121 2.121 0 0 1 3-3h1.5" />
                </svg>
                关于本站
              </h2>
              <p class="text-xs sm:text-sm text-gray-600 leading-relaxed pl-6 sm:pl-7">
                小泉动漫是一个致力于分享二次元二创内容的平台，我们希望为创作者和爱好者提供一个友好的交流空间。在这里，您可以发布自己的作品，发现更多精彩内容，与同好一起交流分享。
              </p>
            </section>

            <section>
              <h2 class="text-base sm:text-lg font-semibold text-gray-800 mb-3 flex items-center gap-2">
                <svg class="w-4 h-4 sm:w-5 sm:h-5 text-green-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                  <polyline points="22 4 12 14.01 9 11.01" />
                </svg>
                我们的宗旨
              </h2>
              <ul class="space-y-2 pl-6 sm:pl-7">
                <li class="text-xs sm:text-sm text-gray-600 flex items-start gap-2">
                  <span class="w-1.5 h-1.5 rounded-full bg-green-500 mt-1.5 shrink-0"></span>
                  <span>尊重创作者，保护知识产权</span>
                </li>
                <li class="text-xs sm:text-sm text-gray-600 flex items-start gap-2">
                  <span class="w-1.5 h-1.5 rounded-full bg-green-500 mt-1.5 shrink-0"></span>
                  <span>提供优质的内容分享平台</span>
                </li>
                <li class="text-xs sm:text-sm text-gray-600 flex items-start gap-2">
                  <span class="w-1.5 h-1.5 rounded-full bg-green-500 mt-1.5 shrink-0"></span>
                  <span>促进二次元文化交流与传播</span>
                </li>
                <li class="text-xs sm:text-sm text-gray-600 flex items-start gap-2">
                  <span class="w-1.5 h-1.5 rounded-full bg-green-500 mt-1.5 shrink-0"></span>
                  <span>打造健康、积极的社区环境</span>
                </li>
              </ul>
            </section>

            <section>
              <h2 class="text-base sm:text-lg font-semibold text-gray-800 mb-3 flex items-center gap-2">
                <svg class="w-4 h-4 sm:w-5 sm:h-5 text-purple-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                  <circle cx="9" cy="7" r="4" />
                  <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
                  <path d="M16 3.13a4 4 0 0 1 0 7.75" />
                </svg>
                联系方式
              </h2>
              <div class="pl-6 sm:pl-7 space-y-2">
                <div class="text-xs sm:text-sm text-gray-600 flex items-center gap-2">
                  <svg class="w-4 h-4 text-gray-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M3 8l7.89 5.26a2 2 0 0 0 2.22 0L21 8M5 19h14a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2z" />
                  </svg>
                  <span>邮箱：contact@xiaoquan.design</span>
                </div>
                <div class="text-xs sm:text-sm text-gray-600 flex items-center gap-2">
                  <svg class="w-4 h-4 text-gray-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M12 20h9" />
                    <path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7.5 18H4v-3.5a2.121 2.121 0 0 1 3-3h1.5" />
                  </svg>
                  <span>反馈：欢迎通过站内反馈功能提交建议</span>
                </div>
              </div>
            </section>

            <section>
              <h2 class="text-base sm:text-lg font-semibold text-gray-800 mb-3 flex items-center gap-2">
                <svg class="w-4 h-4 sm:w-5 sm:h-5 text-orange-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M12 8v4l3 3" />
                  <circle cx="12" cy="12" r="10" />
                </svg>
                更新日志
              </h2>
              <div class="pl-6 sm:pl-7 space-y-3">
                <div class="bg-gray-50/80 rounded-lg p-3 sm:p-4">
                  <div class="flex items-center justify-between mb-2">
                    <span class="text-xs sm:text-sm font-medium text-gray-800">v1.0.0</span>
                    <span class="text-xs text-gray-500">2026年5月</span>
                  </div>
                  <ul class="text-xs text-gray-600 space-y-1">
                    <li>• 平台正式上线</li>
                    <li>• 支持图文、图片、视频、链接等多种内容类型</li>
                    <li>• 实现评论系统和点赞功能</li>
                    <li>• 添加用户注册和登录功能</li>
                  </ul>
                </div>
              </div>
            </section>
          </div>

          <div class="mt-6 sm:mt-8 pt-4 sm:pt-6 border-t border-gray-100">
            <p class="text-xs sm:text-sm text-center text-gray-500">
              © 2026 小泉动漫. All rights reserved.
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>