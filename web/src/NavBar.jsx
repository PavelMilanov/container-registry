function NavBar() {
  return (
    <>
      <div class="header text-2xl">Container Registry</div>
      <nav class="bg-gray-50">
        <div class="max-w-screen-xl px-4 py-3">
          <div class="flex">
            <ul class="flex flex-row font-medium mt-0 space-x-8 rtl:space-x-reverse text-sm">
              <li>
                <a
                  href="/registry"
                  class="text-gray-900 hover:underline"
                  aria-current="page"
                >
                  Реестр
                </a>
              </li>
              <li>
                <a href="/settings" class="text-gray-900 hover:underline">
                  Настройки
                </a>
              </li>
              <li>
                <a href="/logout" class="text-gray-900 hover:underline">
                  Выйти
                </a>
              </li>
            </ul>
          </div>
        </div>
      </nav>
    </>
  );
}

export default NavBar;
