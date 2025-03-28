import { createSignal, onMount, lazy, Show } from "solid-js";
import { A, useNavigate } from "@solidjs/router";
import axios from "axios";
import { showToast } from "./utils/notification";
const AddRegistry = lazy(() => import("./modal/AddRegistry"));
const Delete = lazy(() => import("./modal/Delete"));

const API_URL = window.API_URL;

function Registry() {
  const navigate = useNavigate();
  const [isModalOpen, setModalOpen] = createSignal(false);
  const [registryList, setRegistryList] = createSignal([]);
  const [registry, setRegistry] = createSignal("");

  // const openModal = () => setModalOpen(true);
  // const closeModal = () => setModalOpen(false);
  const submitAddRegistry = async () => {
    setModalOpen(false);
    let token = localStorage.getItem("token");
    const headers = {
      Authorization: `Bearer ${token}`,
    };
    try {
      const response = await axios.post(
        API_URL + `/api/registry/${registry()}`,
        {},
        { headers: headers },
      );
      if (response.status === 201) {
        await getRegistry();
        showToast("Реестр добавлен!");
      }
    } catch (error) {
      if (error.response.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
      } else {
        console.log(error);
        showToast("Ошибка!", "error");
      }
    }
  };
  const newRegistry = (value) => setRegistry(value);

  const [isModalDeleteOpen, setModalDeleteOpen] = createSignal(false);

  let copyText = `${API_URL}`.split("//")[1];

  const openDeleteModal = (item) => {
    setModalDeleteOpen(true);
    setRegistry(item);
  };
  // const closeDeleteModal = () => setModalDeleteOpen(false);
  const submitDelete = async () => {
    setModalDeleteOpen(false);
    let token = localStorage.getItem("token");
    const headers = {
      Authorization: `Bearer ${token}`,
    };
    try {
      const response = await axios.delete(
        API_URL + `/api/registry/${registry()}`,
        { headers: headers },
      );
      if (response.status == 202) {
        showToast("Реестр удален!");
      }
      await getRegistry();
    } catch (error) {
      if (error.response.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
      } else {
        console.error(error);
        showToast("Ошибка!", "error");
      }
    }
  };

  async function getRegistry() {
    let token = localStorage.getItem("token");
    const headers = {
      Authorization: `Bearer ${token}`,
    };
    try {
      const response = await axios.get(API_URL + "/api/registry", {
        headers: headers,
      });
      setRegistryList(response.data.data); // в ответе приходит массив "data"
    } catch (error) {
      if (error.response.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
      } else {
        console.error(error);
        showToast("Ошибка!", "error");
      }
    }
  }

  onMount(async () => {
    await getRegistry();
  });

  return (
    <div class="container">
      <h2>Реестры</h2>
      <div class="copy-container">
        <input type="text" value={copyText} readonly />
      </div>
      <div class="card">
        {/* <button
          type="button"
          onClick={openModal}
          class="text-white bg-blue-900 hover:bg-blue-800 focus:ring-4 focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5 me-2 mb-2 focus:outline-none dark:focus:ring-blue-800"
        >
          Добавить реестр
        </button> */}
        <AddRegistry
          // isOpen={isModalOpen()}
          onNewRegistry={newRegistry}
          // onClose={closeModal}
          onSubmit={submitAddRegistry}
        />
        <Delete
          isOpen={isModalDeleteOpen()}
          message={"Все репозитории и образы Docker реестра будут удалены!"}
          // onClose={closeDeleteModal}
          onSubmit={submitDelete}
        />
        <button
          data-modal-target="authentication-modal"
          data-modal-toggle="authentication-modal"
          class="block text-white bg-blue-900 hover:bg-blue-800 focus:ring-4 focus:outline-none focus:ring-blue-300 font-medium rounded-lg text-sm p-5 py-2.5 text-center"
          type="button"
        >
          Добавить реестр
        </button>
        <div class="relative overflow-x-auto shadow-md sm:rounded-lg">
          <table class="w-full text-base text-left rtl:text-right text-gray-500">
            <thead class="text-xs text-gray-700 uppercase bg-gray-50">
              <tr>
                <th scope="col" class="px-6 py-3">
                  Реестр
                </th>
                <th scope="col" class="px-6 py-3">
                  Создан
                </th>
                <th scope="col" class="px-6 py-3">
                  Размер
                </th>
                <th scope="col" class="px-6 py-3"></th>
              </tr>
            </thead>
            <tbody>
              <For each={registryList()}>
                {(registy, i) => (
                  <tr class="bg-white border-b dark:bg-gray-800 dark:border-gray-700 border-gray-200">
                    <th
                      scope="row"
                      class="px-6 py-4 font-medium text-gray-900 whitespace-nowrap dark:text-white"
                    >
                      <A inactiveClass="" href={registy.Name}>
                        {registy.Name}
                      </A>
                    </th>
                    <td class="px-6 py-4">{registy.Size}</td>
                    <td class="px-6 py-4">{registy.CreatedAt}</td>
                    <td class="px-6 py-4">
                      <button
                        type="button"
                        onClick={() => openDeleteModal(registy.Name)}
                        class="py-2.5 px-5 me-2 mb-2 text-sm font-medium text-gray-900 focus:outline-none bg-white rounded-full border border-gray-200 hover:bg-gray-100 hover:text-blue-700 focus:z-10 focus:ring-4 focus:ring-gray-100 dark:focus:ring-gray-700"
                      >
                        Удалить
                      </button>
                    </td>
                  </tr>
                )}
              </For>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

export default Registry;
