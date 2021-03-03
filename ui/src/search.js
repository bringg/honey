import { Filter, ReferenceArrayInput, SelectArrayInput, TextInput, List, Datagrid, TextField } from 'react-admin';

const SearchFilter = (props) => (
    <Filter {...props}>
        <TextInput label="Filter" source="filter" alwaysOn />
        <ReferenceArrayInput alwaysOn label="Backend" source="backend" reference="backends">
            <SelectArrayInput optionText="name" optionValue="name" />
        </ReferenceArrayInput>
    </Filter>
);

export const SearchList = (props) => (
    <List pagination={null} filters={<SearchFilter />} bulkActionButtons={false} {...props}>
        <Datagrid>
            <TextField source="id" />
            <TextField source="name" />
            <TextField source="backend_name" />
            <TextField source="private_ip" />
            <TextField source="public_ip" />
            <TextField source="status" />
            <TextField source="type" />
        </Datagrid>
    </List>
);