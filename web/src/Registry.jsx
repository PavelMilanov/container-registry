import { createSignal, onMount, lazy } from "solid-js";
import { useNavigate, useLocation } from "@solidjs/router";
import axios from "axios";
import { showAlert } from "./utils/alertService";
import Breadcrumb from "./utils/Breadcrumb";
import RegistryTable from "./utils/RegistryTable";
import NavBar from "./NavBar";
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
        API_URL + `/api/${registry()}`,
        {},
        { headers: headers },
      );
      if (response.status === 201) {
        await getRegistry();
        showAlert("Реестр добавлен!");
      }
    } catch (error) {
      if (error.response.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
      } else {
        console.log(error);
        showAlert(error.response.data.error, "error");
      }
    }
  };

  async function getRegistry() {
    let token = localStorage.getItem("token");
    const headers = {
      Authorization: `Bearer ${token}`,
    };
    try {
      const response = await axios.get(API_URL + "/api/", {
        headers: headers,
      });
      setRegistryList(response.data.data); // в ответе приходит массив "data"
    } catch (error) {
      if (error.response?.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
      } else {
        console.error(error);
        showAlert("Ошибка!", "error");
      }
    }
  }

  onMount(async () => {
    await getRegistry();
  });

  return (
    <>
      <NavBar />
      <div class="container">
        <Breadcrumb path={location.pathname} />
        <div class="card">
          <AddRegistry
            onNewRegistry={newRegistry}
            onSubmit={submitAddRegistry}
          />
          <RegistryTable
            items={registryList()}
            delNotification={() => {
              getRegistry();
            }}
          />
        </div>
      </div>
    </>
  );
}
