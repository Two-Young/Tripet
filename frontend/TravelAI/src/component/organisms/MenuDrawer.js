import {StyleSheet, View, Text, Image} from 'react-native';
import React from 'react';
import {Portal, Drawer, Avatar} from 'react-native-paper';
import colors from '../../theme/colors';
import Modal from 'react-native-modal';
import upperProfile from '../../assets/images/upper_profile.png';
import {navigate} from '../../navigation/RootNavigator';
import AsyncStorage from '@react-native-async-storage/async-storage';
import {useRecoilState, useSetRecoilState} from 'recoil';
import userAtom from '../../recoil/user/user';
import {friendsAtom} from '../../recoil/friends/friends';
import {sessionsAtom} from '../../recoil/session/sessions';
import sessionAtom from '../../recoil/session/session';
import currenciesAtom from '../../recoil/currencies/currencies';
import countriesAtom from '../../recoil/countries/countries';

const MenuDrawer = props => {
  const {visible, setVisible} = props;

  const [user, setUser] = useRecoilState(userAtom);
  const setFriends = useSetRecoilState(friendsAtom);
  const setSessions = useSetRecoilState(sessionsAtom);
  const setSession = useSetRecoilState(sessionAtom);
  const setCurrencies = useSetRecoilState(currenciesAtom);
  const setCountries = useSetRecoilState(countriesAtom);

  const userName = React.useMemo(() => user?.user_info?.username, [user]);
  const userImage = React.useMemo(() => user?.user_info?.profile_image, [user]);

  const onClose = () => {
    setVisible(false);
  };

  const navigateToProfile = () => {
    onClose();
    navigate('Profile');
  };

  const navigateToPeople = () => {
    onClose();
    navigate('MyFriends');
  };

  const logout = () => {
    onClose();
    AsyncStorage.clear();
    setUser(null);
    setFriends([]);
    setSessions([]);
    setSession(null);
    setCurrencies([]);
    setCountries([]);
  };

  const navigateToMySessionRequest = () => {
    onClose();
    navigate('SessionRequests');
  };

  // effects
  React.useEffect(() => {
    if (user === null) {
      navigate('SignIn');
    }
  }, [user]);

  return (
    <Modal
      style={styles.modal}
      isVisible={visible}
      animationIn="slideInRight"
      animationOut="slideOutRight"
      swipeDirection={['right']}
      onSwipeComplete={onClose}
      onBackdropPress={onClose}>
      <Drawer.Section style={styles.drawer}>
        <View style={styles.profileWrapper}>
          <Image style={styles.profileImg} source={upperProfile} />
          <Avatar.Image
            source={{uri: userImage}}
            size={86}
            style={{backgroundColor: colors.primary}}
          />
          <Text style={styles.userNameText}>{userName}</Text>
        </View>
        <Drawer.Item label="Profile" onPress={navigateToProfile} icon="account" />
        <Drawer.Item label="Friends" onPress={navigateToPeople} icon="account-multiple" />
        <Drawer.Item label="Session Requests" onPress={navigateToMySessionRequest} icon="mail" />
        <Drawer.Item label="Logout" onPress={logout} icon="logout" />
      </Drawer.Section>
    </Modal>
  );
};

export default MenuDrawer;

const styles = StyleSheet.create({
  modal: {
    margin: 0,
  },
  drawer: {
    position: 'absolute',
    top: 0,
    bottom: 0,
    right: 0,
    width: '80%',
    height: '100%',
    backgroundColor: colors.white,
    paddingTop: 100,
  },
  drawerItem: {
    color: colors.black,
  },
  profileWrapper: {
    width: '100%',
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 15,
  },
  profileImg: {
    width: 93,
    height: 30.45,
    resizeMode: 'contain',
    position: 'absolute',
    top: 0,
  },
  userNameText: {
    fontSize: 20,
    fontWeight: 'bold',
    color: colors.black,
    marginTop: 10,
    letterSpacing: 0.33,
  },
});
