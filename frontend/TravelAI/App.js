import React from 'react';
// import SplashScreen from 'react-native-splash-screen';
import {NavigationContainer} from '@react-navigation/native';
import {RootNavigator} from './src/navigation/RootNavigator';
import {RecoilRoot} from 'recoil';
import {MD3LightTheme as DefaultTheme, PaperProvider} from 'react-native-paper';
import {AxiosInterceptor} from './src/services/api';
import AsyncStorage from '@react-native-async-storage/async-storage';

const theme = {
  ...DefaultTheme,
  colors: {
    ...DefaultTheme.colors,
    primary: '#0D6EFD',
  },
};

const App = () => {
  /*
  	useEffect(() => {
    SplashScreen.hide();
  	}, []);
  */

  /*
      React.useEffect(() => {
    AsyncStorage.clear();
  }, []);
  */

  return (
    <RecoilRoot>
      <PaperProvider theme={theme}>
        <AxiosInterceptor />
        <NavigationContainer>
          <RootNavigator />
        </NavigationContainer>
      </PaperProvider>
    </RecoilRoot>
  );
};

export default App;
