import {StyleSheet, View, FlatList} from 'react-native';
import React from 'react';
import colors from '../theme/colors';
import {useRecoilValue} from 'recoil';
import currenciesAtom from '../recoil/currencies/currencies';
import {Searchbar} from 'react-native-paper';
import countriesAtom from '../recoil/countries/countries';
import CustomHeader from '../component/molecules/CustomHeader';
import SafeArea from '../component/molecules/SafeArea';
import _ from 'lodash';
import CurrencyListItem from '../component/molecules/CurrencyListItem';
import DismissKeyboard from '../component/molecules/DismissKeyboard';

const defaultCurrencyObject = {
  currency_code: '',
  currency_name: '',
  country_code: '',
  country_symbol: '',
};

const DefaultCurrencyScreen = () => {
  // hooks
  const currencies = useRecoilValue(currenciesAtom);
  const countries = useRecoilValue(countriesAtom);

  // states
  const [defaultCurrency, setDefaultCurrency] = React.useState(defaultCurrencyObject);
  const [searchQuery, setSearchQuery] = React.useState('');

  return (
    <SafeArea>
      <DismissKeyboard>
        <CustomHeader title="Default Currency" useMenu={false} />
      </DismissKeyboard>
      <View style={styles.searchbarWrapper}>
        <Searchbar
          placeholder="Search the country"
          value={searchQuery}
          onChangeText={setSearchQuery}
          style={{
            borderRadius: 8,
            backgroundColor: '#F5F4F6',
          }}
        />
      </View>
      <FlatList
        style={{flex: 1}}
        data={currencies.filter(
          item =>
            item.currency_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
            item.currency_code.toLowerCase().includes(searchQuery.toLowerCase()),
        )}
        renderItem={item => (
          <CurrencyListItem
            item={{
              ...item.item,
              country: countries.find(i => i.country_code === item.item.country_code),
            }}
            checked={_.isEqual(defaultCurrency, item.item)}
            onChecked={() => {
              if (_.isEqual(defaultCurrency, item.item)) {
                setDefaultCurrency(defaultCurrencyObject);
              } else {
                setDefaultCurrency(item.item);
              }
            }}
          />
        )}
      />
    </SafeArea>
  );
};

export default DefaultCurrencyScreen;

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.white,
  },
  searchbarWrapper: {
    marginVertical: 12,
    paddingHorizontal: 20,
  },
});
