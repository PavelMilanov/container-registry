export default function NavBar() {
  return (
    <nav class="bg-gray-50">
      <div class="max-w-screen-xl py-2">
        <div class="flex">
          <ul class="flex flex-row font-medium mt-0 space-x-8 rtl:space-x-reverse text-sm">
            <li>
              <a
                href="/registry"
                class="text-gray-900  hover:bg-blue-50 hover:rounded-sm hover:underline text-base"
                aria-current="page"
              >
                Реестр
              </a>
            </li>
            <li>
              <a
                href="/settings"
                class="text-gray-900 hover:bg-blue-50 hover:underline text-base"
              >
                Настройки
              </a>
            </li>
            <li>
              <a
                href="/logout"
                class="text-gray-900 hover:bg-blue-50 hover:underline text-base"
              >
                Выйти
              </a>
            </li>
          </ul>
        </div>
      </div>
    </nav>
  );
}
