import { createSignal, onMount, lazy, Show } from "solid-js";
import { A, useNavigate, useLocation } from "@solidjs/router";
import axios from "axios";
import { showToast } from "./utils/notification";
import Breadcrumb from "./utils/Breadcrumb";
import Table from "./utils/Table";

const AddRegistry = lazy(() => import("./modal/AddRegistry"));
const Delete = lazy(() => import("./modal/Delete"));

const API_URL = window.API_URL;

function Registry() {
  const navigate = useNavigate();
  const location = useLocation();
  const [isModalOpen, setModalOpen] = createSignal(false);
  const [registryList, setRegistryList] = createSignal([]);
  const [registry, setRegistry] = createSignal("");

  const openModal = () => setModalOpen(true);
  const closeModal = () => setModalOpen(false);
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

  // let copyText = `${API_URL}`.split("//")[1];

  const openDeleteModal = (item) => {
    setModalDeleteOpen(true);
    setRegistry(item);
  };
  const closeDeleteModal = () => setModalDeleteOpen(false);
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
      <Breadcrumb path={location.pathname} />
      <div class="card">
        <button
          type="button"
          onClick={openModal}
          class="text-white bg-main border hover:text-main hover:bg-white hover:border focus:ring-4 focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5 me-2 mb-2"
        >
          Добавить реестр
        </button>
        <AddRegistry
          isOpen={isModalOpen()}
          onNewRegistry={newRegistry}
          onClose={closeModal}
          onSubmit={submitAddRegistry}
        />
        <Delete
          isOpen={isModalDeleteOpen()}
          message={"Все репозитории и образы Docker реестра будут удалены!"}
          onClose={closeDeleteModal}
          onSubmit={submitDelete}
        />
        <Table items={registryList()} />
      </div>
    </div>
  );
}

export default Registry;
