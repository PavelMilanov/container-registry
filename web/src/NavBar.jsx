function NavBar() {
  return (
    <nav>
      <a href="/registry">Реестры</a>
      <a href="/settings">Настройки</a>
      {/* <a href="/logout"> */}
      <button
        data-modal-target="popup-modal"
        data-modal-toggle="popup-modal"
        class="block text-white bg-blue-700 hover:bg-blue-800 focus:ring-4 focus:outline-none focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5 text-center dark:bg-blue-600 dark:hover:bg-blue-700 dark:focus:ring-blue-800"
        type="button"
      >
        Toggle modal
      </button>
      {/* </a> */}
    </nav>
  );
}

export default NavBar;
