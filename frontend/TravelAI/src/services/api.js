import AsyncStorage from '@react-native-async-storage/async-storage';
import axios from 'axios';
import {useRecoilState} from 'recoil';
import userAtom from '../recoil/user/user';
import React from 'react';
import reactotron from 'reactotron-react-native';
import {navigate} from '../navigation/RootNavigator';

export const API_URL_PROD = 'http://43.200.219.71:10375';
export const API_URL_DEBUG = 'http://180.226.155.13:10375/';

const api = axios.create({
  baseURL: API_URL_PROD,
  timeout: 10000,
});

// 응답에서 발생한 401 Unauthorized 에러에 대한 Interceptor

export const AxiosInterceptor = () => {
  const [user, setUser] = useRecoilState(userAtom);

  // Reactotron.log('user : ', user);

  const setAuthHeader = token => {
    // Reactotron.log('token : ', token);
    if (token) {
      api.defaults.headers.common.Authorization = `Bearer ${token}`;
    } else {
      delete api.defaults.headers.common.Authorization;
    }
  };

  const setRefreshTokenHeader = token => {
    if (token) {
      reactotron.log('before refresh token : ', api.defaults.headers.common['X-Refresh-Token']);
      api.defaults.headers.common['X-Refresh-Token'] = token;
      reactotron.log('after refresh token : ', api.defaults.headers.common['X-Refresh-Token']);
    } else {
      delete api.defaults.headers.common['X-Refresh-Token'];
    }
  };

  React.useEffect(() => {
    let interceptor = null;
    console.log('use effect : ', user);
    if (user) {
      reactotron.log('user : ', user);
      setAuthHeader(user?.auth_tokens?.access_token?.token);
      setRefreshTokenHeader(user?.auth_tokens?.refresh_token?.token);

      // 새로운 interceptor 생성
      interceptor = api.interceptors.response.use(
        response => {
          return response;
        },
        async error => {
          const {
            config,
            response: {status, data},
          } = error;
          reactotron.log('error: ', error);
          reactotron.log('status: ', status);
          reactotron.log('data: ', data);
          reactotron.log('config: ', config);
          if (status === 401 && data?.error === 'authorization failed: Token is expired') {
            // reactotron.log('access token expired');
            if (!config._retry) {
              try {
                config._retry = true;
                // reactotron.log('try refresh token');
                const res = await authRefreshToken();
                // reactotron.log('res', res);
                const {access_token, refresh_token} = res;
                // setUser({...user, auth_tokens: {access_token, refresh_token}});
                if (access_token) {
                  // reactotron.log('access token valid');
                  error.config.headers.Authorization = `Bearer ${access_token?.token}`;
                }
                if (refresh_token) {
                  // reactotron.log('refresh token valid');
                  error.config.headers['X-Refresh-Token'] = refresh_token?.token;
                }
                setUser({...user, auth_tokens: {access_token, refresh_token}});
                await AsyncStorage.setItem(
                  'user',
                  JSON.stringify({...user, auth_tokens: {access_token, refresh_token}}),
                );
                return Promise.resolve(api.request(config));
                // return api(config);
              } catch (e) {
                // reactotron.log('refresh token failed');
                setUser(null);
                await AsyncStorage.removeItem('user');
                navigate('SignIn');
                throw e;
              }
            }
            // reactotron.log(res);
            /*
                const {access_token, refresh_token} = res;
                if (access_token) {
                  reactotron.log('access token valid');
                  setUser({...user, auth_tokens: {access_token, refresh_token}});
                  api.defaults.headers.common.Authorization = `Bearer ${access_token}`;
                  return api(config);
                }

                */
          }
          return Promise.reject(error);
        },
      );
    }
    // cleanup 함수
    return () => {
      axios.interceptors.response.eject(interceptor);
      axios.defaults.headers.common.Authorization = null;
      axios.defaults.headers.common['X-Refresh-Token'] = null;
    };
  }, [user]);

  return null;
};

export const getPing = async () => {
  const response = await api.get('/ping');
  return response.data;
};

// auth
export const authGoogleSign = async idToken => {
  try {
    const response = await api.post('/auth/google/sign', {id_token: idToken});
    return response.data;
  } catch (error) {
    throw error;
  }
};

export const authFacebookSign = async accessToken => {
  try {
    const response = await api.post('/auth/facebook/sign', {id_token: accessToken});
    return response.data;
  } catch (error) {
    throw error;
  }
};

export const authNaverSign = async accessToken => {
  try {
    const response = await api.post('/auth/naver/sign', {id_token: accessToken});
    return response.data;
  } catch (error) {
    console.error(error);
    throw error;
  }
};

export const authKakaoSign = async accessToken => {
  try {
    const response = await api.post('/auth/kakao/sign', {id_token: accessToken});
    return response.data;
  } catch (error) {
    throw error;
  }
};

export const authRefreshToken = async refreshToken => {
  try {
    const response = await api.post('/auth/refreshToken');
    return response.data;
  } catch (error) {
    throw error;
  }
};

// platform

export const locateAutoComplete = async query => {
  try {
    const response = await api.post('/platform/locate/auto-complete', {
      input: query,
    });
    return response.data;
  } catch (error) {
    throw error;
  }
};

export const locateDetail = async placeid => {
  try {
    const response = await api.get('/platform/locate/location', {
      params: {
        place_id: placeid,
      },
    });
    return response.data;
  } catch (error) {
    throw error;
  }
};

export const locateDirection = async (origin_place_id, destination_place_id) => {
  try {
    const response = await api.get('/platform/locate/direction', {
      params: {
        origin_place_id,
        destination_place_id,
      },
    });
    return response.data;
  } catch (error) {
    throw error;
  }
};

export const locatePin = async (latitude, longitude) => {
  try {
    const response = await api.get('/platform/locate/pin', {
      params: {
        latitude,
        longitude,
      },
    });
    return response.data;
  } catch (error) {
    throw error;
  }
};

export const locatePlacePhoto = async (reference, max_width) => {
  try {
    const response = await api.get('/platform/locate/place-photo', {
      params: {
        reference,
        max_width,
      },
      responseType: 'arraybuffer',
    });
    reactotron.log('response!! : ', response.data);
    return response.data;
  } catch (error) {
    throw error;
  }
};

export const locateLocation = async place_id => {
  try {
    const response = await api.get('/platform/locate/location', {
      params: {
        place_id,
      },
    });
    return response.data;
  } catch (error) {
    throw error;
  }
};

export const locateCountries = async () => {
  try {
    const response = await api.get('/platform/locate/countries');
    return response.data;
  } catch (error) {
    throw error;
  }
};

// platform - session
export const getSessions = async () => {
  try {
    const response = await api.get('/platform/session');
    return response.data;
  } catch (error) {
    console.error(error);
    throw error;
  }
};

export const createSession = async (country_codes, start_at, end_at) => {
  try {
    const response = await api.put('/platform/session', {
      country_codes,
      start_at,
      end_at,
    });
    return response.data;
  } catch (error) {
    throw error;
  }
};

export const deleteSession = async session_id => {
  try {
    const response = await api.delete('/platform/session', {
      data: {
        session_id,
      },
    });
    return response.data;
  } catch (error) {
    throw error;
  }
};

// platform - schedule
export const getSchedules = async (session_id, day) => {
  try {
    const response = await api.get('/platform/schedule', {
      params: {
        session_id,
        day,
      },
    });
    return response.data;
  } catch (error) {
    throw error;
  }
};

export const createSchedule = async ({session_id, place_id, name, start_at, memo}) => {
  try {
    const response = await api.put('/platform/schedule', {
      session_id,
      place_id,
      name,
      start_at,
      memo,
    });
    return response.data;
  } catch (error) {
    throw error;
  }
};

export const deleteSchedule = async schedule_id => {
  try {
    const response = await api.delete('/platform/schedule', {
      data: {
        schedule_id,
      },
    });
    return response.data;
  } catch (error) {
    throw error;
  }
};

export const updateSchedule = async ({schedule_id, place_id, name, start_at, memo}) => {
  try {
    const response = await api.post('/platform/schedule', {
      schedule_id,
      place_id,
      name,
      start_at,
      memo,
    });
    return response.data;
  } catch (error) {
    throw error;
  }
};

// platform - location
export const getLocations = async session_id => {
  try {
    const response = await api.get('/platform/location', {
      params: {
        session_id,
      },
    });
    return response.data;
  } catch (error) {
    throw error;
  }
};

export const createLocation = async (session_id, place_id) => {
  try {
    const response = await api.put('/platform/location', {
      session_id,
      place_id,
    });
    return response.data;
  } catch (error) {
    throw error;
  }
};

export const deleteLocation = async location_id => {
  try {
    const response = await api.delete('/platform/location', {
      data: {
        location_id,
      },
    });
    return response.data;
  } catch (error) {
    throw error;
  }
};

// platform - budget
export const getBudgetList = async () => {
  try {
    const response = await api.get('/platform/budget/list');
    return response.data;
  } catch (error) {
    throw error;
  }
};

export const createBudget = async (sessiontoken, title, amount, currency) => {
  try {
    const response = await api.post('/platform/budget', {
      sessiontoken,
      title,
      amount,
      currency,
    });
    return response.data;
  } catch (error) {
    throw error;
  }
};
