import { createSignal, onMount, lazy } from "solid-js";
import { A, useNavigate } from "@solidjs/router";
import axios from "axios";
import toast from "solid-toast";

const AddRegistry = lazy(() => import("./modal/AddRegistry"));
const Delete = lazy(() => import("./modal/Delete"));

const API_URL = window.API_URL;

function Registry() {
  const navigate = useNavigate();
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
        toast("Реестр добавлен!", {
          style: {
            "background-color": "#1e3c72",
            color: "white",
          },
          className: "notification",
        });
      }
    } catch (error) {
      console.log(error);
      if (error.response.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
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
        toast("Реестр удален!", {
          style: {
            "background-color": "#1e3c72",
            color: "white",
          },
          className: "notification",
        });
      }
      await getRegistry();
    } catch (error) {
      console.error(error);
      toast("Ошибка!", {
        style: {
          "background-color": "#dc3545",
          color: "white",
        },
        className: "notification",
      });
      if (error.response.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
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
      console.error(error);
      toast("Ошибка!", {
        style: {
          "background-color": "#dc3545",
          color: "white",
        },
        className: "notification",
      });
      if (error.response.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
      }
    }
  }

  onMount(async () => {
    await getRegistry();
  });

  return (
    <div class="container">
      <h2>Репозитории</h2>
      <div class="copy-container">
        <input type="text" value={copyText} readonly />
      </div>
      <div class="card">
        <button class="btn btn-primary" onClick={openModal}>
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
        <table>
          <thead>
            <tr>
              <th>Реестр</th>
              {/* <th>Размер</th> */}
              <th>Создан</th>
              <th>...</th>
            </tr>
          </thead>
          <tbody>
            <For each={registryList()}>
              {(registy, i) => (
                <tr>
                  <td>
                    <div>
                      <A inactiveClass="" href={registy.Name}>
                        {registy.Name}
                      </A>
                    </div>
                  </td>
                  {/* <td>{registy.Size}</td> */}
                  <td>{registy.CreatedAt}</td>
                  <td>
                    <button
                      class="btn btn-secondary"
                      onClick={() => openDeleteModal(registy.Name)}
                    >
                      Удалить реестр
                    </button>
                  </td>
                </tr>
              )}
            </For>
          </tbody>
        </table>
      </div>
    </div>
  );
}

export default Registry;
