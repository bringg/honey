import {
  Filter,
  ReferenceArrayInput,
  SelectArrayInput,
  TextInput,
  List,
  Datagrid,
  TextField,
  UrlField,
  SimpleForm,
  Show,
  SimpleShowLayout,
  useRecordContext,
} from "react-admin";
import * as React from "react";
import Paper from "@mui/material/Paper";
import InputBase from "@mui/material/InputBase";
import IconButton from "@mui/material/IconButton";
import TerminalIcon from "@mui/icons-material/Terminal";
import ReactJson from "react-json-view";
import jp from "jsonpath";
import { useWatch } from "react-hook-form";
import isEmpty from "lodash/isEmpty";

const InstanceFilter = (props) => (
  <Filter {...props}>
    <TextInput label="Filter" source="filter" alwaysOn />
    <ReferenceArrayInput
      alwaysOn
      label="Backend"
      source="backend"
      reference="backends"
    >
      <SelectArrayInput optionText="name" optionValue="name" />
    </ReferenceArrayInput>
  </Filter>
);

const jsonQuery = (raw, path) => {
  if (isEmpty(path)) return raw;

  try {
    return jp.query(raw, path);
  } catch (e) {}

  return [];
};

const PathInput = ({ raw }) => {
  const path = useWatch({ name: "path" });
  return <ReactJson name={false} src={jsonQuery(raw, path)} />;
};

const TunnelField = ({ ip }) => {
  const [user, setUser] = React.useState("root");

  const handleChange = (event) => {
    setUser(event.target.value);
  };

  const record = useRecordContext();

  return (
    <Paper
      sx={{ p: "2px 4px", display: "flex", alignItems: "center", width: 150 }}
    >
      <InputBase
        sx={{ ml: 1, flex: 1 }}
        name="user"
        defaultValue="root"
        placeholder="Username"
        onChange={handleChange}
        inputProps={{ "aria-label": "root" }}
      />
      <IconButton
        href={"/api/v1/createtunnel?user=" + user + "&ip=" + record.private_ip}
        target="_blank"
        sx={{ p: "3px" }}
        aria-label="terminal"
      >
        <TerminalIcon />
      </IconButton>
    </Paper>
  );
};

const InstancePanel = ({ id, record, resource }) => {
  return (
    <div>
      <SimpleForm toolbar={false}>
        <TextInput source="path" label="JsonPath" />
        <PathInput raw={record.raw} />
      </SimpleForm>
    </div>
  );
};

export const InstanceList = (props) => (
  <List filters={<InstanceFilter />} {...props}>
    <Datagrid bulkActionButtons={false} expand={<InstancePanel />}>
      <TextField source="name" />
      <TextField source="backend_name" />
      <TextField source="private_ip" />
      <TextField source="public_ip" />
      <TunnelField source="private_ip" label="Tunnel" />
    </Datagrid>
  </List>
);

export const InstanceShow = (props) => (
  <Show {...props}>
    <SimpleShowLayout>
      <TextField source="name" />
    </SimpleShowLayout>
  </Show>
);
