import { createSignal, onMount } from "solid-js";
import { useNavigate } from "@solidjs/router";
import { showAlert } from "./utils/alertService";
import axios from "axios";
import NavBar from "./NavBar";

export default function Settings() {
  const navigate = useNavigate();
  const [version, setVersion] = createSignal("");
  const [tagCount, setTagCount] = createSignal("");

  async function garbageCollection() {
    let token = localStorage.getItem("token");
    const headers = {
      Authorization: `Bearer ${token}`,
    };
    try {
      const response = await axios.post(
        API_URL + "/api/settings?garbage=true",
        {},
        { headers: headers },
      );
      if (response.status === 202) {
        showAlert(response.data.data);
      } else {
        showAlert(error.response.data.error, "error");
      }
    } catch (error) {
      console.log(error.response.data);
      if (error.response.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
      } else {
        showAlert(error.response.data.error, "error");
      }
    }
  }

  async function setCount() {
    let token = localStorage.getItem("token");
    const headers = {
      Authorization: `Bearer ${token}`,
    };
    try {
      const response = await axios.post(
        API_URL + `/api/settings?tag=${tagCount()}`,
        {},
        { headers: headers },
      );
      if (response.status === 202) {
        showAlert(response.data.data);
      }
    } catch (error) {
      console.log(error.response.data);
      if (error.response.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
      }
    }
  }

  async function getSettings() {
    let token = localStorage.getItem("token");
    const headers = {
      Authorization: `Bearer ${token}`,
    };
    try {
      const response = await axios.get(API_URL + "/api/settings", {
        headers: headers,
      });
      setVersion(response.data.version);
      setTagCount(response.data.count);
    } catch (error) {
      console.log(error.response.data);
      if (error.response.status === 401) {
        localStorage.removeItem("token");
        navigate("/login", { replace: true });
      }
    }
  }

  onMount(async () => {
    await getSettings();
  });

  return (
    <>
      <NavBar />
      <div class="container">
        <h2>Общие настройки</h2>
        <div class="card">
          <div class="relative w-1/4 m-2">
            <input
              type="text"
              id="disabled_filled"
              class="block rounded-t-lg px-2.5 pb-2.5 pt-5 w-full text-sm text-gray-900 bg-gray-50 border-0 border-b-2 border-gray-300 appearance-none focus:outline-none focus:ring-0 focus:border-blue-600 peer"
              placeholder=""
              disabled
            />
            <label
              for="disabled_filled"
              class="absolute text-sm text-gray-400 transform -translate-y-4 scale-75 top-4 z-10 origin-[0] start-2.5 peer-focus:text-blue-600 peer-placeholder-shown:scale-100 peer-placeholder-shown:translate-y-0 peer-focus:scale-75 peer-focus:-translate-y-4 rtl:peer-focus:translate-x-1/4 rtl:peer-focus:left-auto"
            >
              Удалить неиспользуемые файлы реестра:
            </label>
          </div>
          <button
            class="relative text-white bg-main border hover:text-main hover:bg-white hover:border focus:ring-4 focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5 ml-2"
            onClick={garbageCollection}
          >
            Сборщик мусора
          </button>
          <div class="relative w-1/4 m-2 mt-4">
            <input
              type="text"
              id="floating_outlined"
              onInput={(e) => setTagCount(e.target.value)}
              value={tagCount()}
              class="block px-2.5 pb-2.5 pt-4 w-full text-sm text-gray-900 bg-transparent rounded-lg border-1 border-gray-300 appearance-none focus:outline-none focus:ring-0 focus:border-blue-600 peer"
              placeholder=" "
            />
            <label
              for="floating_outlined"
              class="absolute text-sm text-gray-500 duration-300 transform -translate-y-4 scale-75 top-2 z-10 origin-[0] bg-white px-2 peer-focus:px-2 peer-focus:text-blue-600 peer-focus:dark:text-blue-500 peer-placeholder-shown:scale-100 peer-placeholder-shown:-translate-y-1/2 peer-placeholder-shown:top-1/2 peer-focus:top-2 peer-focus:scale-75 peer-focus:-translate-y-4 rtl:peer-focus:translate-x-1/4 rtl:peer-focus:left-auto start-1"
            >
              Удалять теги образов старше:
            </label>
          </div>
          <button
            class="text-white bg-main border hover:text-main hover:bg-white hover:border focus:ring-4 focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5 ml-2"
            onClick={setCount}
          >
            Сохранить настройки
          </button>
          <div class="relative w-1/8 m-2 mt-4">
            <input
              type="text"
              id="disabled_outlined"
              class="block px-2.5 pb-2.5 pt-4 w-full text-sm text-gray-900 bg-transparent rounded-lg border-1 border-gray-300 appearance-none dark:text-white dark:border-gray-600 dark:focus:border-blue-500 focus:outline-none focus:ring-0 focus:border-blue-600 peer"
              placeholder=" "
              disabled
            />
            <label
              for="disabled_outlined"
              class="absolute text-sm text-gray-400 duration-300 transform -translate-y-4 scale-75 top-2 z-10 origin-[0] bg-white px-2 peer-focus:px-2 peer-focus:text-blue-600 peer-focus:dark:text-blue-500 peer-placeholder-shown:scale-100 peer-placeholder-shown:-translate-y-1/2 peer-placeholder-shown:top-1/2 peer-focus:top-2 peer-focus:scale-75 peer-focus:-translate-y-4 start-1 rtl:peer-focus:translate-x-1/4 rtl:peer-focus:left-auto"
            >
              {version()}
            </label>
          </div>
        </div>
      </div>
    </>
  );
}
