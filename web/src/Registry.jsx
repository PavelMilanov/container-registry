import { createSignal, onMount, lazy } from "solid-js";
import { useNavigate, useLocation } from "@solidjs/router";
import axios from "axios";
import { showToast } from "./utils/notification";
import Breadcrumb from "./utils/Breadcrumb";
import RegistryTable from "./utils/RegistryTable";
const AddRegistry = lazy(() => import("./modal/AddRegistry"));

export default function Registry() {
  const navigate = useNavigate();
  const location = useLocation();

  const [registryList, setRegistryList] = createSignal([]);
  const [registry, setRegistry] = createSignal("");

  const newRegistry = (value) => setRegistry(value);
  const submitAddRegistry = async () => {
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
      if (error.response?.status === 401) {
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
        <AddRegistry onNewRegistry={newRegistry} onSubmit={submitAddRegistry} />
        <RegistryTable
          items={registryList()}
          delNotification={() => {
            getRegistry();
          }}
        />
      </div>
    </div>
  );
}
